package service

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"gorm.io/gorm"
)

var (
	ErrCanvasDisabled      = errors.New("canvas integration is disabled")
	ErrCanvasAuthorization = errors.New("canvas authorization is invalid")
)

type CanvasAuthorizeRequest struct {
	RedirectURI         string `json:"redirect_uri"`
	State               string `json:"state"`
	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method"`
}

type CanvasExchangeRequest struct {
	Code         string `json:"code"`
	CodeVerifier string `json:"code_verifier"`
	RedirectURI  string `json:"redirect_uri"`
}

type canvasAuthorization struct {
	Identity      AuthIdentity `json:"identity"`
	RedirectURI   string       `json:"redirect_uri"`
	CodeChallenge string       `json:"code_challenge"`
}

type CanvasExchangeData struct {
	User   CanvasUser   `json:"user"`
	Token  CanvasToken  `json:"token"`
	Wallet CanvasWallet `json:"wallet"`
}

type CanvasUser struct {
	ID          int    `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Status      int    `json:"status"`
	Role        int    `json:"role"`
	Group       string `json:"group"`
}

type CanvasIdentityData struct {
	ID     int `json:"id"`
	Status int `json:"status"`
	Role   int `json:"role"`
}

type CanvasToken struct {
	ID  int    `json:"id"`
	Key string `json:"key"`
}

type CanvasWallet struct {
	Quota            int     `json:"quota"`
	UsedQuota        int     `json:"used_quota"`
	QuotaPerUnit     float64 `json:"quota_per_unit"`
	QuotaDisplayType string  `json:"quota_display_type"`
	USDExchangeRate  float64 `json:"usd_exchange_rate"`
}

func AuthorizeCanvas(identity AuthIdentity, request CanvasAuthorizeRequest) (string, error) {
	origin, _, enabled := common.StolasCanvasConfiguration()
	if !enabled {
		return "", ErrCanvasDisabled
	}
	challenge, err := base64.RawURLEncoding.Strict().DecodeString(request.CodeChallenge)
	if err != nil || len(challenge) != sha256.Size || request.CodeChallengeMethod != "S256" || request.RedirectURI != origin+"/api/auth/stolas/callback" || request.State == "" || len(request.State) > 512 || strings.ContainsAny(request.State, "\r\n\x00") {
		return "", ErrCanvasAuthorization
	}
	if _, pat := model.ParseAccessTokenSessionID(identity.SessionID); pat {
		return "", ErrCanvasAuthorization
	}
	payload, err := common.Marshal(canvasAuthorization{Identity: identity, RedirectURI: request.RedirectURI, CodeChallenge: request.CodeChallenge})
	if err != nil {
		return "", err
	}
	var code string
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		if err := model.ValidateAuthSessionWithTx(tx, identity); err != nil {
			return err
		}
		var createErr error
		code, _, createErr = model.CreateAuthFlowWithTx(tx, model.AuthFlowCreate{
			Purpose: model.AuthFlowPurposeCanvas, UserId: identity.UserID, SessionId: identity.SessionID,
			Payload: string(payload), ExpiresAt: time.Now().Add(time.Minute),
		})
		return createErr
	})
	if err != nil {
		return "", err
	}
	redirect, _ := url.Parse(request.RedirectURI)
	query := url.Values{"code": {code}, "state": {request.State}}
	redirect.RawQuery = query.Encode()
	return redirect.String(), nil
}

func ExchangeCanvas(request CanvasExchangeRequest) (*CanvasExchangeData, error) {
	origin, _, enabled := common.StolasCanvasConfiguration()
	if !enabled {
		return nil, ErrCanvasDisabled
	}
	if request.RedirectURI != origin+"/api/auth/stolas/callback" || len(request.Code) != 43 || len(request.CodeVerifier) < 43 || len(request.CodeVerifier) > 128 {
		return nil, ErrCanvasAuthorization
	}
	for _, character := range request.CodeVerifier {
		if !(character >= 'A' && character <= 'Z' || character >= 'a' && character <= 'z' || character >= '0' && character <= '9' || strings.ContainsRune("-._~", character)) {
			return nil, ErrCanvasAuthorization
		}
	}
	var result CanvasExchangeData
	_, err := model.ConsumeAuthFlowWithAction(request.Code, model.AuthFlowMatch{Purpose: model.AuthFlowPurposeCanvas}, func(tx *gorm.DB, flow *model.AuthFlow) error {
		var authorization canvasAuthorization
		if err := common.UnmarshalJsonStr(flow.Payload, &authorization); err != nil {
			return ErrCanvasAuthorization
		}
		digest := sha256.Sum256([]byte(request.CodeVerifier))
		challenge := base64.RawURLEncoding.EncodeToString(digest[:])
		if authorization.RedirectURI != request.RedirectURI || subtle.ConstantTimeCompare([]byte(challenge), []byte(authorization.CodeChallenge)) != 1 || authorization.Identity.UserID != flow.UserId || authorization.Identity.SessionID != flow.SessionId {
			return ErrCanvasAuthorization
		}
		if _, pat := model.ParseAccessTokenSessionID(authorization.Identity.SessionID); pat {
			return ErrCanvasAuthorization
		}
		if err := model.ValidateAuthSessionWithTx(tx, authorization.Identity); err != nil {
			return err
		}
		user, token, err := model.EnsureCanvasDefaultTokenWithTx(tx, flow.UserId)
		if err != nil {
			return err
		}
		result.User = CanvasUser{ID: user.Id, Username: user.Username, DisplayName: user.DisplayName, Email: user.Email, Status: user.Status, Role: user.Role, Group: user.Group}
		result.Token = CanvasToken{ID: token.Id, Key: "sk-" + token.Key}
		result.Wallet = CanvasWallet{Quota: user.Quota, UsedQuota: user.UsedQuota, QuotaPerUnit: common.QuotaPerUnit, QuotaDisplayType: operation_setting.GetQuotaDisplayType(), USDExchangeRate: operation_setting.USDExchangeRate}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CanvasIdentity reads current authority without creating or replacing a token.
// Both the authenticated token and the explicit default-token association must match.
func CanvasIdentity(userID, tokenID int) (*CanvasIdentityData, error) {
	user, err := model.GetUserById(userID, false)
	if err != nil {
		return nil, err
	}
	if user.Status != common.UserStatusEnabled || user.DefaultTokenID == nil || *user.DefaultTokenID != tokenID {
		return nil, ErrAuthTokenInvalid
	}
	token, err := model.GetTokenById(tokenID)
	if err != nil {
		return nil, err
	}
	if token.UserId != userID || token.Status != common.TokenStatusEnabled || (token.ExpiredTime > 0 && token.ExpiredTime <= time.Now().Unix()) {
		return nil, ErrAuthTokenInvalid
	}
	return &CanvasIdentityData{ID: user.Id, Status: user.Status, Role: user.Role}, nil
}
