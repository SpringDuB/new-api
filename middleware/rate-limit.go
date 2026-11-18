package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

const redisRateLimitNamespace = "rateLimit:v2"

// Redis rate limiting intentionally uses a fixed window. The single Lua script
// makes increment, expiry, and the limit decision atomic, while retaining the
// simple fixed-window behavior: traffic at a window boundary can burst up to
// twice the configured limit. Do not replace this with a sliding-window ZSET
// unless that externally visible behavior is intentionally changed.
const redisFixedWindowScript = `
local count = redis.call('INCR', KEYS[1])
if count == 1 then
  redis.call('EXPIRE', KEYS[1], ARGV[2])
end
local ttl = redis.call('TTL', KEYS[1])
if ttl < 0 then
  redis.call('EXPIRE', KEYS[1], ARGV[2])
  ttl = redis.call('TTL', KEYS[1])
end
if count > tonumber(ARGV[1]) then
  return {0, count, ttl}
end
return {1, count, ttl}
`

var inMemoryRateLimiter common.InMemoryRateLimiter

var defNext = func(c *gin.Context) {
	c.Next()
}

func redisIPRateLimitKey(mark string, clientIP string) string {
	return fmt.Sprintf("%s:ip:%s:%s", redisRateLimitNamespace, mark, clientIP)
}

func redisUserRateLimitKey(mark string, userID int) string {
	return fmt.Sprintf("%s:user:%s:%d", redisRateLimitNamespace, mark, userID)
}

func redisReplyInteger(value any) (int64, error) {
	switch typed := value.(type) {
	case int64:
		return typed, nil
	case string:
		return strconv.ParseInt(typed, 10, 64)
	case []byte:
		return strconv.ParseInt(string(typed), 10, 64)
	default:
		return 0, fmt.Errorf("unexpected Redis integer reply type %T", value)
	}
}

func redisFixedWindowTake(ctx context.Context, key string, maxRequestNum int, duration int64) (bool, int64, int64, error) {
	if common.RDB == nil {
		return false, 0, 0, errors.New("Redis client is not initialized")
	}
	if key == "" {
		return false, 0, 0, errors.New("rate limit key is empty")
	}
	if maxRequestNum <= 0 {
		return false, 0, 0, errors.New("rate limit maximum must be positive")
	}
	if duration <= 0 {
		return false, 0, 0, errors.New("rate limit duration must be positive")
	}

	values, err := common.RDB.Eval(
		ctx,
		redisFixedWindowScript,
		[]string{key},
		maxRequestNum,
		duration,
	).Slice()
	if err != nil {
		return false, 0, 0, err
	}
	if len(values) != 3 {
		return false, 0, 0, fmt.Errorf("unexpected Redis rate limit reply length %d", len(values))
	}

	allowedValue, err := redisReplyInteger(values[0])
	if err != nil {
		return false, 0, 0, err
	}
	count, err := redisReplyInteger(values[1])
	if err != nil {
		return false, 0, 0, err
	}
	ttlSeconds, err := redisReplyInteger(values[2])
	if err != nil {
		return false, 0, 0, err
	}

	return allowedValue == 1, count, ttlSeconds, nil
}

func redisRateLimiter(c *gin.Context, maxRequestNum int, duration int64, mark string) {
	allowed, _, ttlSeconds, err := redisFixedWindowTake(
		c.Request.Context(),
		redisIPRateLimitKey(mark, c.ClientIP()),
		maxRequestNum,
		duration,
	)
	if err != nil {
		logger.LogError(c.Request.Context(), common.LogText("rate limit check failed (mark=%s): %v", mark, err))
		c.Status(http.StatusInternalServerError)
		c.Abort()
		return
	}
	if !allowed {
		writeRateLimited(c, ttlSeconds)
	}
}

func memoryRateLimiter(c *gin.Context, maxRequestNum int, duration int64, mark string) {
	key := mark + c.ClientIP()
	if !inMemoryRateLimiter.Request(key, maxRequestNum, duration) {
		writeRateLimited(c, duration)
		return
	}
}

// writeRateLimited rejects the request with 429 and a Retry-After hint so
// clients can back off instead of treating the rejection as a fatal error.
// The in-memory limiter cannot report the remaining window, so callers
// without a TTL pass the full window duration as a conservative upper bound.
func writeRateLimited(c *gin.Context, retryAfterSeconds int64) {
	if retryAfterSeconds > 0 {
		c.Header("Retry-After", strconv.FormatInt(retryAfterSeconds, 10))
	}
	c.Status(http.StatusTooManyRequests)
	c.Abort()
}

func rateLimitFactory(maxRequestNum int, duration int64, mark string) func(c *gin.Context) {
	if common.RedisEnabled {
		return func(c *gin.Context) {
			redisRateLimiter(c, maxRequestNum, duration, mark)
		}
	}
	// It's safe to call multi times.
	inMemoryRateLimiter.Init(common.RateLimitKeyExpirationDuration)
	return func(c *gin.Context) {
		memoryRateLimiter(c, maxRequestNum, duration, mark)
	}
}

// GlobalWebRateLimit limits dashboard pages and other fallback paths per IP.
// Files that exist in the frontend build count against a separate static
// limiter instead, so lazy-loaded chunks and icons do not use up the page
// budget. The static limiter is disabled by default.
func GlobalWebRateLimit(frontendFS static.ServeFileSystem) func(c *gin.Context) {
	webLimit := defNext
	if common.GlobalWebRateLimitEnable {
		webLimit = rateLimitFactory(common.GlobalWebRateLimitNum, common.GlobalWebRateLimitDuration, "GW")
	}
	staticLimit := defNext
	if common.GlobalStaticRateLimitEnable {
		staticLimit = rateLimitFactory(common.GlobalStaticRateLimitNum, common.GlobalStaticRateLimitDuration, "GS")
	}
	return func(c *gin.Context) {
		if frontendFS.Exists("/", c.Request.URL.Path) {
			staticLimit(c)
			return
		}
		webLimit(c)
	}
}

func GlobalAPIRateLimit() func(c *gin.Context) {
	ordinary := defNext
	if common.GlobalApiRateLimitEnable {
		ordinary = rateLimitFactory(common.GlobalApiRateLimitNum, common.GlobalApiRateLimitDuration, "GA")
	}
	// These exact routes serve many users behind the canvas backend's address.
	// Each still has a bounded ingress bucket, independent of login's CT/GA.
	maximum := max(1, common.GetEnvOrDefault("STOLAS_BRIDGE_INGRESS_RATE_LIMIT", 600))
	usage := rateLimitFactory(maximum, 60, "CANVAS:usage")
	authorize := rateLimitFactory(maximum, 60, "CANVAS:authorize")
	exchange := rateLimitFactory(maximum, 60, "CANVAS:exchange")
	login := rateLimitFactory(maximum, 60, "CANVAS:login")
	verification := rateLimitFactory(maximum, 60, "CANVAS:verification")
	sessionCleanup := rateLimitFactory(maximum, 60, "CANVAS:session-cleanup")
	metadata := rateLimitFactory(maximum, 60, "CANVAS:metadata")
	identity := rateLimitFactory(maximum, 60, "CANVAS:identity-ingress")
	return func(c *gin.Context) {
		switch c.Request.Method + " " + c.FullPath() {
		case "GET /api/usage/token/":
			usage(c)
			return
		case "GET /api/canvas/identity":
			identity(c)
			return
		case "POST /api/canvas/authorize":
			authorize(c)
			return
		case "POST /api/canvas/exchange":
			exchange(c)
			return
		}
		if common.IsStolasCanvasClient(c.GetHeader("X-Stolas-Client-Secret")) {
			switch c.Request.Method + " " + c.FullPath() {
			case "GET /api/status", "GET /api/user/login/encryption-key":
				metadata(c)
				return
			case "POST /api/user/login":
				login(c)
				return
			case "POST /api/user/login/verify", "POST /api/user/login/2fa":
				verification(c)
				return
			case "DELETE /api/user/sessions/:sid":
				sessionCleanup(c)
				return
			}
		}
		ordinary(c)
	}
}

// CanvasLoginRateLimit preserves the public login throttle. Authenticated
// canvas transport uses the dedicated API ingress plus account/flow budgets.
func CanvasLoginRateLimit() gin.HandlerFunc {
	ordinary := CriticalRateLimit()
	return func(c *gin.Context) {
		if common.IsStolasCanvasClient(c.GetHeader("X-Stolas-Client-Secret")) {
			c.Next()
			return
		}
		ordinary(c)
	}
}

// CanvasAuthAttemptRateLimit limits a trusted transport's password identifier,
// resolved account, or verification flow. Identifiers and flow tokens are
// HMACed so neither Redis nor errors reveal authentication material.
func CanvasAuthAttemptRateLimit(c *gin.Context, scope string, userID int, identifier string) bool {
	if !common.CriticalRateLimitEnable || !common.IsStolasCanvasClient(c.GetHeader("X-Stolas-Client-Secret")) {
		return true
	}
	identity := "user:" + strconv.Itoa(userID)
	if userID <= 0 {
		identity = "identifier:" + common.GenerateHMACWithKey([]byte("canvas-login-v1:"+common.SessionSecret), strings.ToLower(strings.TrimSpace(identifier)))
	}
	key := redisRateLimitNamespace + ":canvas-auth:" + scope + ":" + identity
	if common.RedisEnabled {
		userRedisRateLimiter(c, common.CriticalRateLimitNum, common.CriticalRateLimitDuration, key)
		return !c.IsAborted()
	}
	inMemoryRateLimiter.Init(common.RateLimitKeyExpirationDuration)
	if !inMemoryRateLimiter.Request(key, common.CriticalRateLimitNum, common.CriticalRateLimitDuration) {
		writeRateLimited(c, common.CriticalRateLimitDuration)
		return false
	}
	return true
}

// TokenUsageRateLimit must follow TokenAuthReadOnly so untrusted token IDs
// cannot select another user's allowance. Multiple keys share their owner.
func TokenUsageRateLimit() gin.HandlerFunc {
	maximum := max(1, common.GetEnvOrDefault("TOKEN_USAGE_RATE_LIMIT", 120))
	return userRateLimitFactory(maximum, 60, "TOKEN:usage")
}

func CanvasIdentityRateLimit() gin.HandlerFunc {
	maximum := max(1, common.GetEnvOrDefault("STOLAS_IDENTITY_RATE_LIMIT", 600))
	return userRateLimitFactory(maximum, 60, "CANVAS:identity")
}

func CriticalRateLimit() func(c *gin.Context) {
	if common.CriticalRateLimitEnable {
		return rateLimitFactory(common.CriticalRateLimitNum, common.CriticalRateLimitDuration, "CT")
	}
	return defNext
}

func UserCriticalRateLimit(scope string) func(c *gin.Context) {
	if !common.CriticalRateLimitEnable {
		return defNext
	}
	return userRateLimitFactory(
		common.CriticalRateLimitNum,
		common.CriticalRateLimitDuration,
		"UC:"+scope,
	)
}

func DownloadRateLimit() func(c *gin.Context) {
	return rateLimitFactory(common.DownloadRateLimitNum, common.DownloadRateLimitDuration, "DW")
}

func UploadRateLimit() func(c *gin.Context) {
	return rateLimitFactory(common.UploadRateLimitNum, common.UploadRateLimitDuration, "UP")
}

// userRateLimitFactory creates a rate limiter keyed by authenticated user ID
// instead of client IP, making it resistant to proxy rotation attacks.
// Must be used AFTER authentication middleware (UserAuth).
func userRateLimitFactory(maxRequestNum int, duration int64, mark string) func(c *gin.Context) {
	if common.RedisEnabled {
		return func(c *gin.Context) {
			userID := c.GetInt("id")
			if userID == 0 {
				c.Status(http.StatusUnauthorized)
				c.Abort()
				return
			}
			userRedisRateLimiter(c, maxRequestNum, duration, redisUserRateLimitKey(mark, userID))
		}
	}
	// It's safe to call multi times.
	inMemoryRateLimiter.Init(common.RateLimitKeyExpirationDuration)
	return func(c *gin.Context) {
		userID := c.GetInt("id")
		if userID == 0 {
			c.Status(http.StatusUnauthorized)
			c.Abort()
			return
		}
		key := fmt.Sprintf("%s:user:%d", mark, userID)
		if !inMemoryRateLimiter.Request(key, maxRequestNum, duration) {
			writeRateLimited(c, duration)
			return
		}
	}
}

// userRedisRateLimiter is like redisRateLimiter but accepts a pre-built key
// (to support user-ID-based keys).
func userRedisRateLimiter(c *gin.Context, maxRequestNum int, duration int64, key string) {
	allowed, _, ttlSeconds, err := redisFixedWindowTake(c.Request.Context(), key, maxRequestNum, duration)
	if err != nil {
		logger.LogError(c.Request.Context(), common.LogText("rate limit check failed (key=%s): %v", key, err))
		c.Status(http.StatusInternalServerError)
		c.Abort()
		return
	}
	if !allowed {
		writeRateLimited(c, ttlSeconds)
	}
}

// SearchRateLimit returns a per-user rate limiter for search endpoints.
// Configurable via SEARCH_RATE_LIMIT_ENABLE / SEARCH_RATE_LIMIT / SEARCH_RATE_LIMIT_DURATION.
func SearchRateLimit() func(c *gin.Context) {
	if !common.SearchRateLimitEnable {
		return defNext
	}
	return userRateLimitFactory(common.SearchRateLimitNum, common.SearchRateLimitDuration, "SR")
}
