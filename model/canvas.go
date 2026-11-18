package model

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var ErrCanvasDefaultTokenUnavailable = errors.New("canvas default token is unavailable")

// EnsureCanvasDefaultTokenWithTx initializes only users that have never received
// a default token. The association survives token revocation and deletion.
// Callers keep this transaction open until the dependent action commits.
func EnsureCanvasDefaultTokenWithTx(tx *gorm.DB, userID int) (*User, *Token, error) {
	var user User
	if err := lockForUpdate(tx).First(&user, userID).Error; err != nil {
		return nil, nil, err
	}
	if user.Status != common.UserStatusEnabled {
		return nil, nil, ErrUserSessionInactive
	}
	if user.DefaultTokenID == nil {
		key, err := common.GenerateKey()
		if err != nil {
			return nil, nil, err
		}
		token := Token{
			UserId: user.Id, Name: "Stolas default", Key: key,
			Status: common.TokenStatusEnabled, CreatedTime: common.GetTimestamp(),
			AccessedTime: common.GetTimestamp(), ExpiredTime: -1,
			UnlimitedQuota: true, AllowUserQuotaQuery: true,
		}
		if setting.DefaultUseAutoGroup {
			token.Group = "auto"
		}
		// The token key must stay out of SQL logs even when debug logging is on.
		if err := tx.Session(&gorm.Session{Logger: logger.Discard}).Create(&token).Error; err != nil {
			return nil, nil, err
		}
		result := tx.Model(&User{}).Where("id = ? AND default_token_id IS NULL", user.Id).Update("default_token_id", token.Id)
		if result.Error != nil {
			return nil, nil, result.Error
		}
		if result.RowsAffected != 1 {
			return nil, nil, ErrCanvasDefaultTokenUnavailable
		}
		user.DefaultTokenID = &token.Id
		return &user, &token, nil
	}
	var token Token
	if err := lockForUpdate(tx).Where("id = ? AND user_id = ?", *user.DefaultTokenID, user.Id).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrCanvasDefaultTokenUnavailable
		}
		return nil, nil, err
	}
	if token.Key == "" || token.Status != common.TokenStatusEnabled || (token.ExpiredTime != -1 && token.ExpiredTime <= common.GetTimestamp()) || (!token.UnlimitedQuota && token.RemainQuota <= 0) {
		return nil, nil, ErrCanvasDefaultTokenUnavailable
	}
	return &user, &token, nil
}
