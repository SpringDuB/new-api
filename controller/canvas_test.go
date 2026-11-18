package controller

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/oauth"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func canvasDatabase(t *testing.T, kind, dsn string) *gorm.DB {
	t.Helper()
	db, _ := newAuditTestDatabase(t, kind, dsn)
	db = db.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})
	previousDB, previousLogDB, previousRedis := model.DB, model.LOG_DB, common.RedisEnabled
	previousType := common.MainDatabaseType()
	model.DB, model.LOG_DB, common.RedisEnabled = db, db, false
	if kind == "mysql" {
		common.SetMainDatabaseType(common.DatabaseTypeMySQL)
	} else if kind == "postgres" {
		common.SetMainDatabaseType(common.DatabaseTypePostgreSQL)
	} else {
		common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	}
	t.Cleanup(func() {
		connection, err := db.DB()
		if err == nil {
			_ = connection.Close()
		}
		model.DB, model.LOG_DB, common.RedisEnabled = previousDB, previousLogDB, previousRedis
		common.SetMainDatabaseType(previousType)
	})
	t.Setenv("STOLAS_CANVAS_URL", "https://canvas.stolas.cc")
	t.Setenv("STOLAS_CANVAS_CLIENT_SECRET", "test-canvas-secret-with-at-least-32-characters")
	for range 2 {
		require.NoError(t, db.AutoMigrate(&model.User{}, &model.Token{}, &model.AuthFlow{}, &model.UserSession{}))
	}
	versionQuery := "SELECT version()"
	if kind == "sqlite" {
		versionQuery = "SELECT sqlite_version()"
	}
	var version string
	require.NoError(t, db.Raw(versionQuery).Scan(&version).Error)
	t.Logf("%s engine: %s", kind, version)
	return db
}

func canvasRequest(t *testing.T, handler gin.HandlerFunc, bearer string, input any) *httptest.ResponseRecorder {
	t.Helper()
	body, err := common.Marshal(input)
	require.NoError(t, err)
	response := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(response)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/canvas", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Authorization", "Bearer "+bearer)
	handler(c)
	return response
}

func canvasUserSession(t *testing.T, db *gorm.DB) (*model.User, string) {
	t.Helper()
	user := &model.User{Username: "creator", Email: "creator@example.com", DisplayName: "Creator", Status: common.UserStatusEnabled, Role: common.RoleCommonUser, Group: "default", AuthVersion: 1, AffCode: "creator", Quota: 1234567, UsedQuota: 12345}
	require.NoError(t, db.Create(user).Error)
	session := model.UserSession{SID: "canvas-browser-session", UserID: user.Id, Version: 1, UserAuthVersion: 1, Status: model.UserSessionStatusActive, ExpiresAt: time.Now().Add(time.Hour).Unix()}
	require.NoError(t, db.Create(&session).Error)
	token, _, err := service.IssueAccessToken(service.AuthIdentity{UserID: user.Id, SessionID: session.SID, UserAuthVersion: 1, SessionVersion: 1})
	require.NoError(t, err)
	return user, token
}

func canvasAuthorizeCode(t *testing.T, bearer string) string {
	t.Helper()
	response := canvasRequest(t, CanvasAuthorize, bearer, map[string]any{
		"redirect_uri": "https://canvas.stolas.cc/api/auth/stolas/callback", "state": "browser-bound-state",
		"code_challenge": "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM", "code_challenge_method": "S256",
	})
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var result struct {
		Data struct {
			RedirectURL string `json:"redirect_url"`
		}
	}
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
	redirect, err := url.Parse(result.Data.RedirectURL)
	require.NoError(t, err)
	assert.Equal(t, "https", redirect.Scheme)
	assert.Equal(t, "canvas.stolas.cc", redirect.Host)
	assert.Equal(t, "/api/auth/stolas/callback", redirect.Path)
	assert.Equal(t, "browser-bound-state", redirect.Query().Get("state"))
	assert.Len(t, redirect.Query(), 2, "redirect contains only code and browser state")
	assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
	require.NotEmpty(t, redirect.Query().Get("code"))
	return redirect.Query().Get("code")
}

func canvasExchangeInput(code string) map[string]string {
	return map[string]string{"code": code, "code_verifier": "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk", "redirect_uri": "https://canvas.stolas.cc/api/auth/stolas/callback"}
}

func TestCanvasIdentityReadsCurrentRoleAndRequiresBridgeAndDefaultAssociation(t *testing.T) {
	for _, dialect := range []struct{ kind, env string }{{"sqlite", ""}, {"mysql", "TEST_MYSQL_DSN"}, {"postgres", "TEST_POSTGRES_DSN"}} {
		t.Run(dialect.kind, func(t *testing.T) {
			if dialect.env != "" && os.Getenv(dialect.env) == "" {
				t.Skip("set " + dialect.env + " to run this database")
			}
			db := canvasDatabase(t, dialect.kind, os.Getenv(dialect.env))
			user, _ := canvasUserSession(t, db)
			var token *model.Token
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				_, initialized, err := model.EnsureCanvasDefaultTokenWithTx(tx, user.Id)
				token = initialized
				return err
			}))
			engine := gin.New()
			verifiedTokenID := token.Id
			engine.GET("/api/canvas/identity", func(c *gin.Context) {
				c.Set("id", user.Id)
				c.Set("token_id", verifiedTokenID)
				CanvasIdentity(c)
			})
			read := func(secret, key string) *httptest.ResponseRecorder {
				request := httptest.NewRequest(http.MethodGet, "/api/canvas/identity", nil)
				request.Header.Set("Authorization", "Bearer "+key)
				request.Header.Set("X-Stolas-Client-Secret", secret)
				response := httptest.NewRecorder()
				engine.ServeHTTP(response, request)
				return response
			}
			secret := os.Getenv("STOLAS_CANVAS_CLIENT_SECRET")
			assert.Equal(t, http.StatusUnauthorized, read("", "sk-"+token.Key).Code)
			verifiedTokenID = token.Id + 999
			assert.Equal(t, http.StatusUnauthorized, read(secret, "sk-"+token.Key).Code)
			verifiedTokenID = token.Id
			for _, role := range []int{common.RoleAdminUser, common.RoleRootUser, common.RoleCommonUser} {
				require.NoError(t, db.Model(&model.User{}).Where("id = ?", user.Id).Update("role", role).Error)
				response := read(secret, "sk-"+token.Key)
				require.Equal(t, http.StatusOK, response.Code, response.Body.String())
				var result struct {
					Data service.CanvasUser `json:"data"`
				}
				require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
				var fields struct {
					Data map[string]any `json:"data"`
				}
				require.NoError(t, common.Unmarshal(response.Body.Bytes(), &fields))
				assert.Len(t, fields.Data, 3, "identity exposes only id, status and role")
				assert.Equal(t, role, result.Data.Role)
				assert.Equal(t, user.Id, result.Data.ID)
				assert.NotContains(t, response.Body.String(), token.Key)
				assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
			}
			require.NoError(t, db.Model(&model.Token{}).Where("id = ?", token.Id).Update("expired_time", time.Now().Add(-time.Minute).Unix()).Error)
			assert.Equal(t, http.StatusUnauthorized, read(secret, "sk-"+token.Key).Code)
			require.NoError(t, db.Model(&model.Token{}).Where("id = ?", token.Id).Update("expired_time", -1).Error)
			require.NoError(t, db.Model(&model.User{}).Where("id = ?", user.Id).Update("status", common.UserStatusDisabled).Error)
			assert.Contains(t, []int{http.StatusUnauthorized, http.StatusForbidden}, read(secret, "sk-"+token.Key).Code)
		})
	}
}

func TestCanvasAuthorizationSecurity(t *testing.T) {
	for _, tc := range []string{"success and replay", "wrong verifier", "wrong callback", "wrong client", "expired", "revoked session", "auth version changed", "disabled user", "revoked key", "deleted key", "cross user key", "PAT rejected", "disabled config"} {
		t.Run(tc, func(t *testing.T) {
			db := canvasDatabase(t, "sqlite", "")
			user, bearer := canvasUserSession(t, db)
			if tc == "PAT rejected" {
				response := canvasRequest(t, CanvasAuthorize, "management-pat", map[string]string{})
				assert.Equal(t, http.StatusUnauthorized, response.Code)
				return
			}
			if tc == "disabled config" {
				t.Setenv("STOLAS_CANVAS_CLIENT_SECRET", "too-short")
				response := canvasRequest(t, CanvasAuthorize, bearer, map[string]string{})
				assert.Equal(t, http.StatusServiceUnavailable, response.Code)
				return
			}
			code := canvasAuthorizeCode(t, bearer)
			input, secret := canvasExchangeInput(code), os.Getenv("STOLAS_CANVAS_CLIENT_SECRET")
			switch tc {
			case "wrong verifier":
				input["code_verifier"] = strings.Repeat("x", 43)
			case "wrong callback":
				input["redirect_uri"] += "?evil=true"
			case "wrong client":
				secret = "wrong-client-secret"
			case "expired":
				require.NoError(t, db.Model(&model.AuthFlow{}).Where("user_id = ?", user.Id).Update("expires_at", time.Now().Add(-time.Minute)).Error)
			case "revoked session":
				require.NoError(t, db.Model(&model.UserSession{}).Where("user_id = ?", user.Id).Update("revoked_at", time.Now().Unix()).Error)
			case "auth version changed":
				require.NoError(t, db.Model(user).Update("auth_version", 2).Error)
			case "disabled user":
				require.NoError(t, db.Model(user).Update("status", common.UserStatusDisabled).Error)
			case "revoked key", "deleted key", "cross user key":
				require.NoError(t, db.Transaction(func(tx *gorm.DB) error { _, _, err := model.EnsureCanvasDefaultTokenWithTx(tx, user.Id); return err }))
				var token model.Token
				require.NoError(t, db.Where("user_id = ?", user.Id).First(&token).Error)
				if tc == "deleted key" {
					require.NoError(t, db.Delete(&token).Error)
				} else if tc == "cross user key" {
					require.NoError(t, db.Model(&token).Update("user_id", user.Id+100).Error)
				} else {
					require.NoError(t, db.Model(&token).Update("status", common.TokenStatusDisabled).Error)
				}
			}
			response := canvasRequest(t, CanvasExchange, secret, input)
			assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
			if tc != "success and replay" {
				assert.NotEqual(t, http.StatusOK, response.Code, response.Body.String())
				assert.NotContains(t, response.Body.String(), `"key":`)
				if tc == "wrong verifier" || tc == "wrong callback" || tc == "wrong client" {
					retry := canvasRequest(t, CanvasExchange, os.Getenv("STOLAS_CANVAS_CLIENT_SECRET"), canvasExchangeInput(code))
					assert.Equal(t, http.StatusOK, retry.Code, retry.Body.String())
				}
				return
			}
			require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			var result struct {
				Data struct {
					User struct {
						ID       int
						Username string
					}
					Token struct {
						ID  int
						Key string
					}
					Wallet struct {
						Quota        int
						UsedQuota    int     `json:"used_quota"`
						QuotaPerUnit float64 `json:"quota_per_unit"`
					}
				}
			}
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
			assert.Equal(t, user.Id, result.Data.User.ID)
			assert.Equal(t, user.Username, result.Data.User.Username)
			assert.Equal(t, 1234567, result.Data.Wallet.Quota)
			assert.Equal(t, 12345, result.Data.Wallet.UsedQuota)
			assert.Positive(t, result.Data.Wallet.QuotaPerUnit)
			assert.True(t, strings.HasPrefix(result.Data.Token.Key, "sk-"))
			assert.NotContains(t, response.Body.String(), "password")
			assert.NotContains(t, response.Body.String(), "auth_version")
			replay := canvasRequest(t, CanvasExchange, secret, input)
			assert.NotEqual(t, http.StatusOK, replay.Code)
		})
	}
}

func TestCanvasExchangeDatabaseMatrix(t *testing.T) {
	for _, dialect := range []struct{ kind, env string }{{"sqlite", ""}, {"mysql", "TEST_MYSQL_DSN"}, {"postgres", "TEST_POSTGRES_DSN"}} {
		t.Run(dialect.kind, func(t *testing.T) {
			if dialect.env != "" && os.Getenv(dialect.env) == "" {
				t.Skip("set " + dialect.env + " to run this database")
			}
			db := canvasDatabase(t, dialect.kind, os.Getenv(dialect.env))
			_, bearer := canvasUserSession(t, db)
			code := canvasAuthorizeCode(t, bearer)
			statuses := make(chan int, 2)
			var workers sync.WaitGroup
			for range 2 {
				workers.Go(func() {
					response := canvasRequest(t, CanvasExchange, os.Getenv("STOLAS_CANVAS_CLIENT_SECRET"), canvasExchangeInput(code))
					statuses <- response.Code
				})
			}
			workers.Wait()
			close(statuses)
			successes := 0
			for status := range statuses {
				if status == http.StatusOK {
					successes++
				}
			}
			assert.Equal(t, 1, successes, "authorization code must have exactly one successful consumer")
			var count int64
			require.NoError(t, db.Model(&model.Token{}).Count(&count).Error)
			assert.EqualValues(t, 1, count, "concurrent exchange cannot create duplicate defaults")
		})
	}
}

func TestCanvasRegistrationInitializesDefaultToken(t *testing.T) {
	for _, dialect := range []struct{ kind, env string }{{"sqlite", ""}, {"mysql", "TEST_MYSQL_DSN"}, {"postgres", "TEST_POSTGRES_DSN"}} {
		t.Run(dialect.kind, func(t *testing.T) {
			if dialect.env != "" && os.Getenv(dialect.env) == "" {
				t.Skip("set " + dialect.env + " to run this database")
			}
			db := canvasDatabase(t, dialect.kind, os.Getenv(dialect.env))
			user := model.User{Username: "new-creator", DisplayName: "Creator", Status: common.UserStatusEnabled, Role: common.RoleCommonUser}
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error { return user.InsertWithTx(tx, 0) }))
			var tokens []model.Token
			require.NoError(t, db.Where("user_id = ?", user.Id).Find(&tokens).Error)
			require.Len(t, tokens, 1, "registration must create its default model credential atomically")
			assert.True(t, tokens[0].UnlimitedQuota)
			assert.True(t, tokens[0].AllowUserQuotaQuery)
			assert.False(t, tokens[0].ModelLimitsEnabled)
			assert.Equal(t, int64(-1), tokens[0].ExpiredTime)
		})
	}
}

func TestCanvasDefaultTokenLifecycle(t *testing.T) {
	db := canvasDatabase(t, "sqlite", "")
	previousMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(previousMode) })
	t.Run("password registration is atomic and does not duplicate legacy initialization", func(t *testing.T) {
		previousGenerate, previousEmail, previousRegister, previousPassword := constant.GenerateDefaultToken, common.EmailVerificationEnabled, common.RegisterEnabled, common.PasswordRegisterEnabled
		constant.GenerateDefaultToken, common.EmailVerificationEnabled, common.RegisterEnabled, common.PasswordRegisterEnabled = true, false, true, true
		t.Cleanup(func() {
			constant.GenerateDefaultToken, common.EmailVerificationEnabled, common.RegisterEnabled, common.PasswordRegisterEnabled = previousGenerate, previousEmail, previousRegister, previousPassword
		})
		response := canvasRequest(t, Register, "", map[string]string{"username": "password-creator", "password": "secure-test-password"})
		require.Equal(t, http.StatusOK, response.Code)
		var result struct{ Success bool }
		require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
		require.True(t, result.Success, response.Body.String())
		var user model.User
		require.NoError(t, db.Where("username = ?", "password-creator").First(&user).Error)
		require.NotNil(t, user.DefaultTokenID)
		var count int64
		require.NoError(t, db.Model(&model.Token{}).Where("user_id = ?", user.Id).Count(&count).Error)
		assert.EqualValues(t, 1, count)
	})
	t.Run("token initialization failure rolls registration back", func(t *testing.T) {
		require.NoError(t, db.Callback().Create().Before("gorm:create").Register("canvas_test_token_failure", func(tx *gorm.DB) {
			if tx.Statement.Table == "tokens" {
				tx.AddError(errors.New("test token persistence failure"))
			}
		}))
		t.Cleanup(func() { require.NoError(t, db.Callback().Create().Remove("canvas_test_token_failure")) })
		for _, transactional := range []bool{false, true} {
			user := model.User{Username: "rollback-creator", Status: common.UserStatusEnabled, Role: common.RoleCommonUser}
			var err error
			if transactional {
				err = db.Transaction(func(tx *gorm.DB) error { return user.InsertWithTx(tx, 0) })
			} else {
				err = user.Insert(0)
			}
			require.Error(t, err)
			var count int64
			require.NoError(t, db.Model(&model.User{}).Where("username = ?", "rollback-creator").Count(&count).Error)
			assert.Zero(t, count)
		}
	})
	t.Run("OAuth registration initializes the associated credential", func(t *testing.T) {
		previousRegister := common.RegisterEnabled
		common.RegisterEnabled = true
		t.Cleanup(func() { common.RegisterEnabled = previousRegister })
		response := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(response)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/oauth/test", nil)
		user, _, err := findOrCreateOAuthUser(c, &authFlowTestOAuthProvider{}, &oauth.OAuthUser{ProviderUserID: "oauth-creator", Username: "oauth-creator", DisplayName: "OAuth Creator"}, &oauth.OAuthToken{}, "")
		require.NoError(t, err)
		require.NotNil(t, user.DefaultTokenID)
		var token model.Token
		require.NoError(t, db.Where("id = ? AND user_id = ?", *user.DefaultTokenID, user.Id).First(&token).Error)
		assert.True(t, token.UnlimitedQuota)
		assert.True(t, token.AllowUserQuotaQuery)
	})
	t.Run("incomplete config leaves registration unchanged", func(t *testing.T) {
		t.Setenv("STOLAS_CANVAS_CLIENT_SECRET", "")
		user := model.User{Username: "ordinary-user", Status: common.UserStatusEnabled, Role: common.RoleCommonUser}
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error { return user.InsertWithTx(tx, 0) }))
		var count int64
		require.NoError(t, db.Model(&model.Token{}).Where("user_id = ?", user.Id).Count(&count).Error)
		assert.Zero(t, count)
		assert.Nil(t, user.DefaultTokenID)
	})
}

func TestCanvasAuthorizeRejectsInvalidClientParameters(t *testing.T) {
	db := canvasDatabase(t, "sqlite", "")
	_, bearer := canvasUserSession(t, db)
	for _, tc := range []struct{ name, field, value string }{
		{"plain PKCE", "code_challenge_method", "plain"},
		{"wrong challenge length", "code_challenge", "abcd"},
		{"missing state", "state", ""},
		{"cross origin redirect", "redirect_uri", "https://evil.example/api/auth/stolas/callback"},
		{"callback query", "redirect_uri", "https://canvas.stolas.cc/api/auth/stolas/callback?next=evil"},
		{"callback fragment", "redirect_uri", "https://canvas.stolas.cc/api/auth/stolas/callback#evil"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := map[string]string{"redirect_uri": "https://canvas.stolas.cc/api/auth/stolas/callback", "state": "browser-bound-state", "code_challenge": "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM", "code_challenge_method": "S256"}
			input[tc.field] = tc.value
			response := canvasRequest(t, CanvasAuthorize, bearer, input)
			assert.Equal(t, http.StatusBadRequest, response.Code)
		})
	}
	var count int64
	require.NoError(t, db.Model(&model.AuthFlow{}).Count(&count).Error)
	assert.Zero(t, count, "invalid requests must not create redeemable authorizations")
}

func TestCanvasAuthorizeOnlyAcceptsDashboardBearer(t *testing.T) {
	db := canvasDatabase(t, "sqlite", "")
	require.NoError(t, db.AutoMigrate(&model.UserAccessToken{}, &model.AuditLog{}))
	user, bearer := canvasUserSession(t, db)
	pat, _ := createScopedAccessToken(t, user.Id, 0, "api_key:reveal")
	router := gin.New()
	router.POST("/api/canvas/authorize", middleware.UserAuth(), CanvasAuthorize)
	input := `{"redirect_uri":"https://canvas.stolas.cc/api/auth/stolas/callback","state":"browser-bound-state","code_challenge":"E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM","code_challenge_method":"S256"}`
	for _, tc := range []struct {
		name, authorization string
		want                int
	}{
		{"dashboard bearer", "Bearer " + bearer, http.StatusOK},
		{"valid management PAT", "Bearer " + pat, http.StatusForbidden},
		{"cookie only", "", http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/canvas/authorize", strings.NewReader(input))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", tc.authorization)
			request.AddCookie(&http.Cookie{Name: service.RefreshCookieName, Value: "canvas-browser-session.cookie"})
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			assert.Equal(t, tc.want, response.Code, response.Body.String())
		})
	}
	var flow model.AuthFlow
	require.NoError(t, db.Where("purpose = ?", model.AuthFlowPurposeCanvas).First(&flow).Error)
	assert.WithinDuration(t, time.Now().Add(time.Minute), flow.ExpiresAt, 5*time.Second, "authorization must expire after one minute")
}

func TestCanvasDefaultCredentialNeverAppearsInSQLLogs(t *testing.T) {
	db := canvasDatabase(t, "sqlite", "")
	user := model.User{Username: "logging-creator", AffCode: "logging", Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(&user).Error)
	var output bytes.Buffer
	verbose := db.Session(&gorm.Session{Logger: logger.New(log.New(&output, "", 0), logger.Config{LogLevel: logger.Info})})
	var token *model.Token
	require.NoError(t, verbose.Transaction(func(tx *gorm.DB) error {
		var err error
		_, token, err = model.EnsureCanvasDefaultTokenWithTx(tx, user.Id)
		return err
	}))
	require.NotNil(t, token)
	assert.NotContains(t, output.String(), token.Key, "credential creation must suppress plaintext SQL parameters even in debug mode")
}

func TestCanvasConfigurationPublicStatus(t *testing.T) {
	for _, tc := range []struct {
		name, origin, secret string
		enabled              bool
		wantOrigin           string
	}{
		{"configured", "https://canvas.stolas.cc/", strings.Repeat("s", 32), true, "https://canvas.stolas.cc"},
		{"missing secret", "https://canvas.stolas.cc", "", false, ""},
		{"short secret", "https://canvas.stolas.cc", "short", false, ""},
		{"http", "http://canvas.stolas.cc", strings.Repeat("s", 32), false, ""},
		{"userinfo", "https://evil@canvas.stolas.cc", strings.Repeat("s", 32), false, ""},
		{"path", "https://canvas.stolas.cc/path", strings.Repeat("s", 32), false, ""},
		{"query", "https://canvas.stolas.cc?x=y", strings.Repeat("s", 32), false, ""},
		{"fragment", "https://canvas.stolas.cc#", strings.Repeat("s", 32), false, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("STOLAS_CANVAS_URL", tc.origin)
			t.Setenv("STOLAS_CANVAS_CLIENT_SECRET", tc.secret)
			response := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(response)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/status", nil)
			GetStatus(c)
			var result struct {
				Data struct {
					Enabled bool   `json:"canvas_enabled"`
					URL     string `json:"canvas_url"`
				}
			}
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
			assert.Equal(t, tc.enabled, result.Data.Enabled)
			assert.Equal(t, tc.wantOrigin, result.Data.URL)
			if tc.secret != "" {
				assert.NotContains(t, response.Body.String(), tc.secret)
			}
		})
	}
}

func TestCanvasDefaultTokenMigrationDatabaseMatrix(t *testing.T) {
	for _, dialect := range []struct{ kind, env string }{{"sqlite", ""}, {"mysql", "TEST_MYSQL_DSN"}, {"postgres", "TEST_POSTGRES_DSN"}} {
		t.Run(dialect.kind, func(t *testing.T) {
			if dialect.env != "" && os.Getenv(dialect.env) == "" {
				t.Skip("set " + dialect.env + " to run this database")
			}
			db, _ := newAuditTestDatabase(t, dialect.kind, os.Getenv(dialect.env))
			connection, err := db.DB()
			require.NoError(t, err)
			t.Cleanup(func() { _ = connection.Close() })
			require.NoError(t, db.AutoMigrate(&releasedAuditUser{}, &model.Token{}))
			// rc.50 adds this persisted field to the rc.33 user fixture above;
			// its remaining differences are validation or ignored Go-only fields.
			require.NoError(t, db.Migrator().AddColumn(&model.User{}, "AccessTokenCreatedAt"))
			legacy := releasedAuditUser{Username: "legacy-creator", DisplayName: "Legacy", AffCode: "legacy", Status: common.UserStatusEnabled, AuthVersion: 1, Group: "default"}
			require.NoError(t, db.Create(&legacy).Error)
			existing := model.Token{UserId: legacy.Id, Key: "legacy-model-key", Name: "Stolas default", UnlimitedQuota: false, RemainQuota: 321, ModelLimitsEnabled: true, ModelLimits: "one-model", ExpiredTime: -1}
			require.NoError(t, db.Create(&existing).Error)
			for range 2 {
				require.NoError(t, db.AutoMigrate(&model.User{}, &model.Token{}))
			}
			var upgraded model.User
			require.NoError(t, db.First(&upgraded, legacy.Id).Error)
			assert.Nil(t, upgraded.DefaultTokenID, "old tokens are never guessed from names or ordering")
			previousDB, previousType := model.DB, common.MainDatabaseType()
			model.DB = db
			if dialect.kind == "mysql" {
				common.SetMainDatabaseType(common.DatabaseTypeMySQL)
			} else if dialect.kind == "postgres" {
				common.SetMainDatabaseType(common.DatabaseTypePostgreSQL)
			} else {
				common.SetMainDatabaseType(common.DatabaseTypeSQLite)
			}
			t.Cleanup(func() { model.DB = previousDB; common.SetMainDatabaseType(previousType) })
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error { _, _, err := model.EnsureCanvasDefaultTokenWithTx(tx, legacy.Id); return err }))
			require.NoError(t, db.First(&upgraded, legacy.Id).Error)
			require.NotNil(t, upgraded.DefaultTokenID)
			assert.NotEqual(t, existing.Id, *upgraded.DefaultTokenID)
			var preserved model.Token
			require.NoError(t, db.First(&preserved, existing.Id).Error)
			assert.Equal(t, "legacy-model-key", preserved.Key)
			assert.Equal(t, 321, preserved.RemainQuota)
			assert.True(t, preserved.ModelLimitsEnabled)
			assert.False(t, preserved.AllowUserQuotaQuery)
			assert.False(t, preserved.UnlimitedQuota)
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error { _, _, err := model.EnsureCanvasDefaultTokenWithTx(tx, legacy.Id); return err }))
			var count int64
			require.NoError(t, db.Model(&model.Token{}).Count(&count).Error)
			assert.EqualValues(t, 2, count)
			duplicateToken := model.Token{UserId: legacy.Id, Key: existing.Key}
			assert.Error(t, db.Session(&gorm.Session{Logger: logger.Discard}).Create(&duplicateToken).Error, "migration must preserve model-key uniqueness")
			duplicateAssociation := model.User{Username: "duplicate-assoc", AffCode: "duplicate-assoc", DefaultTokenID: upgraded.DefaultTokenID}
			assert.Error(t, db.Session(&gorm.Session{Logger: logger.Discard}).Create(&duplicateAssociation).Error, "one default credential cannot be associated with two users")
		})
	}
}
