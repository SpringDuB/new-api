package router

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func canvasRateLimitRouter(t *testing.T, useRedis bool) (*gin.Engine, *model.User, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	previousDB, previousLogDB, previousRedis, previousClient := model.DB, model.LOG_DB, common.RedisEnabled, common.RDB
	previousType := common.MainDatabaseType()
	previousSQLitePath, previousMaster := common.SQLitePath, common.IsMasterNode
	common.SQLitePath, common.IsMasterNode = t.TempDir()+"/canvas-rates.db", false
	t.Setenv("SQL_DSN", "")
	require.NoError(t, model.InitDB())
	db := model.DB
	connection, err := db.DB()
	require.NoError(t, err)
	connection.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Token{}, &model.UserSession{}, &model.AuthFlow{}, &model.AuditLog{}, &model.UserAccessToken{}))
	previousGlobal, previousGlobalNum, previousGlobalDuration := common.GlobalApiRateLimitEnable, common.GlobalApiRateLimitNum, common.GlobalApiRateLimitDuration
	previousCritical, previousCriticalNum, previousCriticalDuration := common.CriticalRateLimitEnable, common.CriticalRateLimitNum, common.CriticalRateLimitDuration
	previousPassword, previousEncryption, previousTurnstile := common.PasswordLoginEnabled, common.PasswordLoginEncryptionEnabled, common.TurnstileCheckEnabled
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	model.DB, model.LOG_DB, common.RedisEnabled, common.RDB = db, db, useRedis, redisClient
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	common.GlobalApiRateLimitEnable, common.GlobalApiRateLimitNum, common.GlobalApiRateLimitDuration = true, 5, 60
	common.CriticalRateLimitEnable, common.CriticalRateLimitNum, common.CriticalRateLimitDuration = true, 2, 1200
	common.PasswordLoginEnabled, common.PasswordLoginEncryptionEnabled, common.TurnstileCheckEnabled = true, false, false
	t.Cleanup(func() {
		_ = connection.Close()
		_ = redisClient.Close()
		model.DB, model.LOG_DB, common.RedisEnabled, common.RDB = previousDB, previousLogDB, previousRedis, previousClient
		common.SQLitePath, common.IsMasterNode = previousSQLitePath, previousMaster
		common.SetMainDatabaseType(previousType)
		common.GlobalApiRateLimitEnable, common.GlobalApiRateLimitNum, common.GlobalApiRateLimitDuration = previousGlobal, previousGlobalNum, previousGlobalDuration
		common.CriticalRateLimitEnable, common.CriticalRateLimitNum, common.CriticalRateLimitDuration = previousCritical, previousCriticalNum, previousCriticalDuration
		common.PasswordLoginEnabled, common.PasswordLoginEncryptionEnabled, common.TurnstileCheckEnabled = previousPassword, previousEncryption, previousTurnstile
	})
	t.Setenv("STOLAS_CANVAS_URL", "https://canvas.stolas.cc")
	t.Setenv("STOLAS_CANVAS_CLIENT_SECRET", "router-test-canvas-secret-at-least-32-characters")
	user := &model.User{Username: "canvas-limit-user", Email: "creator@example.com", Password: "invalid-password-hash", Status: common.UserStatusEnabled, Role: common.RoleCommonUser, Group: "default", AuthVersion: 1, AffCode: "canvas-limit", Quota: 500000}
	require.NoError(t, db.Create(user).Error)
	require.NoError(t, db.Create(&model.Token{UserId: user.Id, Key: "canvasusagekey", Status: common.TokenStatusEnabled, ExpiredTime: -1, UnlimitedQuota: true, AllowUserQuotaQuery: true}).Error)
	require.NoError(t, db.Create(&model.UserSession{SID: "canvas-rate-session", UserID: user.Id, Version: 1, UserAuthVersion: 1, Status: model.UserSessionStatusActive, ExpiresAt: time.Now().Add(time.Hour).Unix()}).Error)
	bearer, _, err := service.IssueAccessToken(service.AuthIdentity{UserID: user.Id, SessionID: "canvas-rate-session", UserAuthVersion: 1, SessionVersion: 1})
	require.NoError(t, err)
	engine := gin.New()
	require.NoError(t, engine.SetTrustedProxies(nil))
	SetApiRouter(engine)
	return engine, user, bearer
}

func canvasRateRequest(t *testing.T, router http.Handler, method, path, bearer, clientSecret string, input any) *httptest.ResponseRecorder {
	t.Helper()
	body, err := common.Marshal(input)
	require.NoError(t, err)
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	request.RemoteAddr = "192.0.2.130:12345"
	request.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		request.Header.Set("Authorization", "Bearer "+bearer)
	}
	if clientSecret != "" {
		request.Header.Set("X-Stolas-Client-Secret", clientSecret)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestCanvasIdentityRouteRequiresModelKeyAndBridgeSecret(t *testing.T) {
	engine, user, _ := canvasRateLimitRouter(t, false)
	var token model.Token
	require.NoError(t, model.DB.First(&token, "user_id = ?", user.Id).Error)
	require.NoError(t, model.DB.Model(&model.User{}).Where("id = ?", user.Id).Updates(map[string]any{"role": common.RoleAdminUser, "default_token_id": token.Id}).Error)
	secret := "router-test-canvas-secret-at-least-32-characters"
	assert.Equal(t, http.StatusUnauthorized, canvasRateRequest(t, engine, "GET", "/api/canvas/identity", "sk-"+token.Key, "", nil).Code)
	assert.Equal(t, http.StatusUnauthorized, canvasRateRequest(t, engine, "GET", "/api/canvas/identity", "wrong-model-key", secret, nil).Code)
	response := canvasRateRequest(t, engine, "GET", "/api/canvas/identity", "sk-"+token.Key, secret, nil)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.Contains(t, response.Body.String(), `"role":10`)
	assert.NotContains(t, response.Body.String(), token.Key)
}

func TestCanvasCreditMigrationRouteRequiresBridgeSecretInBothHeaders(t *testing.T) {
	engine, user, dashboardBearer := canvasRateLimitRouter(t, true)
	const secret = "router-test-canvas-secret-at-least-32-characters"
	input := model.CanvasCreditMigration{ID: strings.Repeat("a", 64), UserID: user.Id, CNYMicros: 1_000_000}
	for _, credentials := range []struct{ name, bearer, header string }{
		{"no credentials", "", ""},
		{"missing custom header", secret, ""},
		{"missing bearer", "", secret},
		{"dashboard bearer", dashboardBearer, secret},
		{"incorrect custom header", secret, "wrong-secret"},
	} {
		t.Run(credentials.name, func(t *testing.T) {
			response := canvasRateRequest(t, engine, http.MethodPost, "/api/canvas/credit-migration", credentials.bearer, credentials.header, input)
			assert.Equal(t, http.StatusUnauthorized, response.Code, response.Body.String())
		})
	}
}

func TestCanvasIdentityAndWalletHaveIndependentUserBudgets(t *testing.T) {
	t.Setenv("TOKEN_USAGE_RATE_LIMIT", "1")
	t.Setenv("STOLAS_IDENTITY_RATE_LIMIT", "2")
	t.Setenv("STOLAS_MODEL_CATALOG_RATE_LIMIT", "1")
	engine, user, _ := canvasRateLimitRouter(t, true)
	var token model.Token
	require.NoError(t, model.DB.First(&token, "user_id = ?", user.Id).Error)
	require.NoError(t, model.DB.Model(&model.User{}).Where("id = ?", user.Id).Update("default_token_id", token.Id).Error)
	secret := "router-test-canvas-secret-at-least-32-characters"
	for range 2 {
		response := canvasRateRequest(t, engine, "GET", "/api/canvas/identity", "sk-"+token.Key, secret, nil)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	}
	assert.Equal(t, http.StatusTooManyRequests, canvasRateRequest(t, engine, "GET", "/api/canvas/identity", "sk-"+token.Key, secret, nil).Code)
	assert.NotEqual(t, http.StatusTooManyRequests, canvasRateRequest(t, engine, "GET", "/api/canvas/model-catalog", "sk-"+token.Key, secret, nil).Code)
	assert.Equal(t, http.StatusTooManyRequests, canvasRateRequest(t, engine, "GET", "/api/canvas/model-catalog", "sk-"+token.Key, secret, nil).Code)
	assert.Equal(t, http.StatusOK, canvasRateRequest(t, engine, "GET", "/api/usage/token/", "sk-"+token.Key, "", nil).Code)
}

func TestCanvasIdentityIngressDoesNotSpendOtherBridgeBudgets(t *testing.T) {
	t.Setenv("STOLAS_BRIDGE_INGRESS_RATE_LIMIT", "2")
	engine, user, _ := canvasRateLimitRouter(t, true)
	var token model.Token
	require.NoError(t, model.DB.First(&token, "user_id = ?", user.Id).Error)
	require.NoError(t, model.DB.Model(&model.User{}).Where("id = ?", user.Id).Update("default_token_id", token.Id).Error)
	secret := "router-test-canvas-secret-at-least-32-characters"
	for range 2 {
		require.Equal(t, http.StatusOK, canvasRateRequest(t, engine, "GET", "/api/canvas/identity", "sk-"+token.Key, secret, nil).Code)
	}
	assert.Equal(t, http.StatusTooManyRequests, canvasRateRequest(t, engine, "GET", "/api/canvas/identity", "sk-"+token.Key, secret, nil).Code)
	assert.Equal(t, http.StatusOK, canvasRateRequest(t, engine, "GET", "/api/status", "", secret, nil).Code)
}

func TestCanvasUsageDoesNotSpendAuthenticationRateBudget(t *testing.T) {
	router, _, bearer := canvasRateLimitRouter(t, true)
	for range 21 {
		response := canvasRateRequest(t, router, http.MethodGet, "/api/usage/token/", "sk-canvasusagekey", "", nil)
		require.Equal(t, http.StatusOK, response.Code, "authenticated wallet requests from the BFF must not use CT/GA: %s", response.Body.String())
	}
	authorize := canvasRateRequest(t, router, http.MethodPost, "/api/canvas/authorize", bearer, "", map[string]string{"redirect_uri": "https://canvas.stolas.cc/api/auth/stolas/callback", "state": "browser-state", "code_challenge": "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM", "code_challenge_method": "S256"})
	require.Equal(t, http.StatusOK, authorize.Code, authorize.Body.String())
	var data struct {
		Data struct {
			RedirectURL string `json:"redirect_url"`
		}
	}
	require.NoError(t, common.Unmarshal(authorize.Body.Bytes(), &data))
	callback, err := url.Parse(data.Data.RedirectURL)
	require.NoError(t, err)
	exchange := canvasRateRequest(t, router, http.MethodPost, "/api/canvas/exchange", "router-test-canvas-secret-at-least-32-characters", "", map[string]string{"code": callback.Query().Get("code"), "code_verifier": "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk", "redirect_uri": "https://canvas.stolas.cc/api/auth/stolas/callback"})
	require.Equal(t, http.StatusOK, exchange.Code, exchange.Body.String())
	for range 2 {
		response := canvasRateRequest(t, router, http.MethodPost, "/api/user/login", "", "", map[string]string{"username": "unknown", "password": "wrong"})
		require.Equal(t, http.StatusOK, response.Code, "usage and bridge traffic must preserve the ordinary login budget")
	}
	response := canvasRateRequest(t, router, http.MethodPost, "/api/user/login", "", "", map[string]string{"username": "unknown", "password": "wrong"})
	assert.Equal(t, http.StatusTooManyRequests, response.Code, "ordinary login abuse must still be throttled")
}

func TestCanvasTrustedLoginRateLimitsKeepAccountAndPublicBounds(t *testing.T) {
	router, user, _ := canvasRateLimitRouter(t, true)
	const secret = "router-test-canvas-secret-at-least-32-characters"
	for i := range 7 {
		response := canvasRateRequest(t, router, http.MethodPost, "/api/user/login", "", secret, map[string]string{"username": fmt.Sprintf("unrelated-user-%d", i), "password": "wrong"})
		require.Equal(t, http.StatusOK, response.Code, "independent BFF accounts must not aggregate into CT/GA")
	}
	for _, username := range []string{user.Username, user.Email} {
		response := canvasRateRequest(t, router, http.MethodPost, "/api/user/login", "", secret, map[string]string{"username": username, "password": "wrong"})
		require.Equal(t, http.StatusOK, response.Code)
	}
	response := canvasRateRequest(t, router, http.MethodPost, "/api/user/login", "", secret, map[string]string{"username": user.Username, "password": "wrong"})
	assert.Equal(t, http.StatusTooManyRequests, response.Code, "username/email attempts must share the same account throttle")
	for range 2 {
		response := canvasRateRequest(t, router, http.MethodPost, "/api/user/login", "", "incorrect-client-secret", map[string]string{"username": "public-unknown", "password": "wrong"})
		require.Equal(t, http.StatusOK, response.Code)
	}
	response = canvasRateRequest(t, router, http.MethodPost, "/api/user/login", "", "incorrect-client-secret", map[string]string{"username": "public-unknown", "password": "wrong"})
	assert.Equal(t, http.StatusTooManyRequests, response.Code, "an untrusted header must never bypass public login throttling")
	for range 2 {
		response := canvasRateRequest(t, router, http.MethodPost, "/api/user/login/verify", "", secret, map[string]string{"flow_token": "invalid-verification-flow", "code": "000000", "method": "2fa"})
		require.NotEqual(t, http.StatusTooManyRequests, response.Code)
	}
	response = canvasRateRequest(t, router, http.MethodPost, "/api/user/login/verify", "", secret, map[string]string{"flow_token": "invalid-verification-flow", "code": "000000", "method": "2fa"})
	assert.Equal(t, http.StatusTooManyRequests, response.Code, "the BFF verification flow still has an attempt budget")
	common.TurnstileCheckEnabled = true
	response = canvasRateRequest(t, router, http.MethodPost, "/api/user/login", "", secret, map[string]string{"username": "captcha-user", "password": "wrong"})
	assert.Contains(t, response.Body.String(), "Turnstile token is empty", "trusted transport must preserve the CAPTCHA requirement")
}

func TestCanvasUsageRetainsIngressAndPerUserRateLimits(t *testing.T) {
	t.Run("anonymous ingress is bounded", func(t *testing.T) {
		t.Setenv("STOLAS_BRIDGE_INGRESS_RATE_LIMIT", "3")
		router, _, _ := canvasRateLimitRouter(t, true)
		for range 3 {
			response := canvasRateRequest(t, router, http.MethodGet, "/api/usage/token/", "", "", nil)
			require.Equal(t, http.StatusUnauthorized, response.Code)
		}
		response := canvasRateRequest(t, router, http.MethodGet, "/api/usage/token/", "", "", nil)
		assert.Equal(t, http.StatusTooManyRequests, response.Code)
		assert.NotEmpty(t, response.Header().Get("Retry-After"))
	})
	t.Run("authenticated users have independent query budgets", func(t *testing.T) {
		t.Setenv("TOKEN_USAGE_RATE_LIMIT", "3")
		router, user, _ := canvasRateLimitRouter(t, true)
		other := model.User{Username: "other-usage-user", AffCode: "other-usage", Status: common.UserStatusEnabled, Role: common.RoleCommonUser, Group: "default", AuthVersion: 1}
		require.NoError(t, model.DB.Create(&other).Error)
		require.NoError(t, model.DB.Create(&model.Token{UserId: other.Id, Key: "otherusagekey", Status: common.TokenStatusEnabled, ExpiredTime: -1, UnlimitedQuota: true}).Error)
		require.NoError(t, model.DB.Create(&model.Token{UserId: user.Id, Key: "sameownerkey", Status: common.TokenStatusEnabled, ExpiredTime: -1, UnlimitedQuota: true}).Error)
		for range 3 {
			response := canvasRateRequest(t, router, http.MethodGet, "/api/usage/token/", "sk-canvasusagekey", "", nil)
			require.Equal(t, http.StatusOK, response.Code)
		}
		response := canvasRateRequest(t, router, http.MethodGet, "/api/usage/token/", "sk-canvasusagekey", "", nil)
		assert.Equal(t, http.StatusTooManyRequests, response.Code)
		response = canvasRateRequest(t, router, http.MethodGet, "/api/usage/token/", "sk-sameownerkey", "", nil)
		assert.Equal(t, http.StatusTooManyRequests, response.Code, "multiple keys must not multiply one user's budget")
		response = canvasRateRequest(t, router, http.MethodGet, "/api/usage/token/", "sk-otherusagekey", "", nil)
		assert.Equal(t, http.StatusOK, response.Code, "another user's budget must be unaffected")
	})
}

func TestCanvasInMemoryRateLimitsPreserveTheSameBoundaries(t *testing.T) {
	router, user, _ := canvasRateLimitRouter(t, false)
	for range 21 {
		response := canvasRateRequest(t, router, http.MethodGet, "/api/usage/token/", "sk-canvasusagekey", "", nil)
		require.Equal(t, http.StatusOK, response.Code)
	}
	for _, username := range []string{user.Username, user.Email} {
		response := canvasRateRequest(t, router, http.MethodPost, "/api/user/login", "", "router-test-canvas-secret-at-least-32-characters", map[string]string{"username": username, "password": "wrong"})
		require.Equal(t, http.StatusOK, response.Code)
	}
	response := canvasRateRequest(t, router, http.MethodPost, "/api/user/login", "", "router-test-canvas-secret-at-least-32-characters", map[string]string{"username": user.Username, "password": "wrong"})
	assert.Equal(t, http.StatusTooManyRequests, response.Code)
	for range 2 {
		response := canvasRateRequest(t, router, http.MethodPost, "/api/user/login", "", "", map[string]string{"username": "ordinary-user", "password": "wrong"})
		require.Equal(t, http.StatusOK, response.Code)
	}
	response = canvasRateRequest(t, router, http.MethodPost, "/api/user/login", "", "", map[string]string{"username": "ordinary-user", "password": "wrong"})
	assert.Equal(t, http.StatusTooManyRequests, response.Code)
	assert.Equal(t, "1200", response.Header().Get("Retry-After"), "the public CT bound remains independent")
}

func TestCanvasTrustedMetadataDoesNotSharePublicGlobalAllowance(t *testing.T) {
	router, _, _ := canvasRateLimitRouter(t, true)
	for range 5 {
		response := canvasRateRequest(t, router, http.MethodGet, "/api/status", "", "", nil)
		require.Equal(t, http.StatusOK, response.Code)
	}
	response := canvasRateRequest(t, router, http.MethodGet, "/api/status", "", "", nil)
	require.Equal(t, http.StatusTooManyRequests, response.Code, "public metadata must retain its ordinary ingress limit")
	for range 7 {
		for _, path := range []string{"/api/status", "/api/user/login/encryption-key"} {
			response := canvasRateRequest(t, router, http.MethodGet, path, "", "router-test-canvas-secret-at-least-32-characters", nil)
			require.Equal(t, http.StatusOK, response.Code, "authenticated BFF metadata must not aggregate into public GA")
		}
	}
	response = canvasRateRequest(t, router, http.MethodGet, "/api/user/login/encryption-key", "", "", nil)
	assert.Equal(t, http.StatusTooManyRequests, response.Code)
}

func TestCanvasTemporarySessionCleanupKeepsAuthenticationAndIndependentIngress(t *testing.T) {
	router, _, bearer := canvasRateLimitRouter(t, true)
	for range 5 {
		response := canvasRateRequest(t, router, http.MethodGet, "/api/status", "", "", nil)
		require.Equal(t, http.StatusOK, response.Code)
	}
	const secret = "router-test-canvas-secret-at-least-32-characters"
	response := canvasRateRequest(t, router, http.MethodDelete, "/api/user/sessions/canvas-rate-session", "", secret, nil)
	require.Equal(t, http.StatusUnauthorized, response.Code, "client authentication alone must not revoke a user session")
	response = canvasRateRequest(t, router, http.MethodDelete, "/api/user/sessions/canvas-rate-session", bearer, secret, nil)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var session model.UserSession
	require.NoError(t, model.DB.Where("sid = ?", "canvas-rate-session").First(&session).Error)
	assert.NotZero(t, session.RevokedAt)
}

// accessTokenExemptRoutes are /api routes that never reach dashboard
// authentication, so a personal access token has no rule to satisfy there.
var accessTokenExemptRoutes = []string{
	// Anonymous pages and setup.
	"GET /api/setup",
	"POST /api/setup",
	"GET /api/status",
	"GET /api/uptime/status",
	"GET /api/notice",
	"GET /api/user-agreement",
	"GET /api/privacy-policy",
	"GET /api/about",
	"GET /api/home_page_content",
	"GET /api/ratio_config",
	"GET /api/user/groups",

	// Anonymous account entry points.
	"GET /api/verification",
	"GET /api/reset_password",
	"POST /api/user/reset",
	"POST /api/user/register",
	"GET /api/user/login/encryption-key",
	"POST /api/user/login",
	"POST /api/user/login/2fa",
	"POST /api/user/login/verify",
	"POST /api/user/login/passkey/begin",
	"POST /api/user/login/passkey/finish",
	"POST /api/user/passkey/login/begin",
	"POST /api/user/passkey/login/finish",
	"GET /api/oauth/wechat",
	"GET /api/oauth/telegram/login",
	"GET /api/oauth/telegram/bind/:flow_token",

	// Browser session cookie endpoints.
	"POST /api/user/auth/refresh",
	"POST /api/user/auth/logout",

	// The canvas backend authenticates with its configured application secret.
	"POST /api/canvas/exchange",
	"GET /api/canvas/identity",
	"GET /api/canvas/model-catalog",
	"POST /api/canvas/credit-migration",

	// Payment callbacks.
	"POST /api/stripe/webhook",
	"POST /api/creem/webhook",
	"POST /api/waffo/webhook",
	"POST /api/waffo-pancake/webhook/:env",
	"GET /api/user/epay/notify",
	"POST /api/user/epay/notify",
	"GET /api/subscription/epay/notify",
	"POST /api/subscription/epay/notify",
	"GET /api/subscription/epay/return",
	"POST /api/subscription/epay/return",

	// API key authentication (TokenAuthReadOnly).
	"GET /api/usage/token/",
	"GET /api/log/token",
}

// accessTokenHelperRoutes are declared through handlePermissionRoute outside
// channelPermissionRoutes.
var accessTokenHelperRoutes = map[string]authz.Permission{
	"GET /api/task_plugin_options": authz.TaskPluginBind,
	"GET /api/audit":               authz.AuditRead,
}

func newAccessTokenScopeTestEngine(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	// A permission route that is also in the static table panics here.
	require.NotPanics(t, func() {
		SetApiRouter(engine)
		SetDashboardRouter(engine)
		SetRelayRouter(engine)
		SetTaskPluginProtocolRouter(engine)
		SetVideoRouter(engine)
		SetTaskRouter(engine)
	})
	return engine
}

func TestAccessTokenRouteRulesCoverEveryDashboardRoute(t *testing.T) {
	engine := newAccessTokenScopeTestEngine(t)
	exempt := map[string]bool{}
	for _, key := range accessTokenExemptRoutes {
		exempt[key] = true
	}
	registered := map[string]bool{}
	panel := map[string]bool{}
	for _, route := range engine.Routes() {
		if !strings.HasPrefix(route.Path, "/api/") && !strings.HasPrefix(route.Path, "/pg/") {
			continue
		}
		key := route.Method + " " + route.Path
		registered[key] = true
		if !exempt[key] {
			panel[key] = true
		}
	}
	for _, key := range accessTokenExemptRoutes {
		assert.True(t, registered[key], "exempt route %s is no longer registered", key)
	}

	declared := map[string]int{}
	for _, key := range middleware.AccessTokenRouteRuleKeys() {
		declared[key]++
	}
	for key := range panel {
		assert.Equal(t, 1, declared[key], "panel route %s must be declared exactly once", key)
	}
	for key := range declared {
		assert.True(t, panel[key], "declared route %s is not a registered panel route", key)
	}

	permissionRoutes := map[string]authz.Permission{}
	for _, route := range channelPermissionRoutes {
		permissionRoutes[route.method+" "+joinPaths("/api/channel", route.path)] = route.permission
	}
	for key, permission := range accessTokenHelperRoutes {
		permissionRoutes[key] = permission
	}
	for key, permission := range permissionRoutes {
		rule, ok := middleware.AccessTokenRouteRule(key)
		require.True(t, ok, key)
		assert.Equal(t, "scope", rule.Kind(), key)
		assert.Equal(t, service.AccessTokenScopeOf(permission), rule.Scope(), key)
	}
}

func TestAccessTokenCatalogCoversEveryPermission(t *testing.T) {
	catalog := service.AccessTokenCatalog(1, common.RoleRootUser)
	scopes := map[string]bool{}
	resources := map[string]int{}
	for _, group := range catalog {
		for _, resource := range group.Resources {
			resources[resource.Resource]++
			for _, action := range resource.Actions {
				scopes[resource.Resource+":"+action.Action] = true
			}
		}
	}
	for _, permission := range authz.AllPermissions() {
		assert.True(t, scopes[service.AccessTokenScopeOf(permission)], "root catalog lacks %s", service.AccessTokenScopeOf(permission))
	}
	for resource, count := range resources {
		assert.Equal(t, 1, count, "resource %s appears in more than one catalog entry", resource)
	}
}

func TestAccessTokenVerificationScopesAreAllListed(t *testing.T) {
	files, err := filepath.Glob("../service/*.go")
	require.NoError(t, err)
	fileSet := token.NewFileSet()
	var scopes []string
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(fileSet, file, nil, 0)
		require.NoError(t, err)
		for _, declaration := range parsed.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.CONST {
				continue
			}
			for _, spec := range general.Specs {
				value := spec.(*ast.ValueSpec)
				for i, name := range value.Names {
					if !strings.HasPrefix(name.Name, "VerificationScope") || i >= len(value.Values) {
						continue
					}
					literal, ok := value.Values[i].(*ast.BasicLit)
					require.True(t, ok, name.Name)
					scope, err := strconv.Unquote(literal.Value)
					require.NoError(t, err)
					scopes = append(scopes, scope)
				}
			}
		}
	}
	require.NotEmpty(t, scopes)
	for _, scope := range scopes {
		_, listed := service.AccessTokenVerificationScope(scope)
		assert.True(t, listed, "verification scope %s has no access token rule", scope)
	}
}
