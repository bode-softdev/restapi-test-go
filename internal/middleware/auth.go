package middleware

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"restapi-test/internal/response"
	"restapi-test/internal/service"
)

const (
	ctxUserID   = "auth_user_id"
	ctxUsername = "auth_username"
	ctxRole     = "auth_role"
)

func Auth(auth *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, ok := extractToken(c)
		if !ok {
			response.Fail(c, 401, response.CodeUnauthorized, "token autentikasi wajib disertakan", map[string]any{
				"hint": "kirim header Authorization: Bearer <authentication_token>",
			})
			return
		}
		claims, err := auth.ParseAccessToken(raw)
		if err != nil {
			code := response.CodeTokenInvalid
			if err == service.ErrTokenExpired {
				code = response.CodeTokenExpired
			}
			response.Fail(c, 401, code, err.Error(), nil)
			return
		}
		c.Set(ctxUserID, claims.UserID)
		c.Set(ctxUsername, claims.Username)
		c.Set(ctxRole, claims.Role)
		c.Next()
	}
}

func CurrentUser(c *gin.Context) (service.Actor, bool) {
	raw, ok := c.Get(ctxUserID)
	if !ok {
		return service.Actor{}, false
	}
	id, ok := raw.(uint)
	if !ok {
		return service.Actor{}, false
	}
	username, _ := c.Get(ctxUsername)
	name, _ := username.(string)
	return service.Actor{ID: id, Username: name}, true
}

func extractToken(c *gin.Context) (string, bool) {
	header := c.GetHeader("Authorization")
	if header != "" {
		parts := strings.SplitN(header, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			if token := strings.TrimSpace(parts[1]); token != "" {
				return token, true
			}
		}
		return "", false
	}
	if token := strings.TrimSpace(c.GetHeader("X-API-Token")); token != "" {
		return token, true
	}
	return "", false
}

type RateLimitRule struct {
	Limit  int
	Window time.Duration
}

func RateLimit(client *redis.Client, prefix, ruleName string, rule RateLimitRule) gin.HandlerFunc {
	return func(c *gin.Context) {
		if client == nil {
			c.Next()
			return
		}
		identifier := c.ClientIP()
		if actor, ok := CurrentUser(c); ok {
			identifier = "user:" + strconv.FormatUint(uint64(actor.ID), 10)
		}

		key := strings.Join([]string{prefix, ruleName, identifier}, ":")
		ctx := c.Request.Context()

		count, err := client.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}
		if count == 1 {
			if err := client.Expire(ctx, key, rule.Window).Err(); err != nil {
				c.Next()
				return
			}
		}

		ttl, err := client.TTL(ctx, key).Result()
		if err != nil || ttl <= 0 {
			ttl = rule.Window
		}
		retryAfter := int(ttl.Seconds())
		if retryAfter < 1 {
			retryAfter = 1
		}
		c.Header("X-RateLimit-Limit", strconv.Itoa(rule.Limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(max(0, rule.Limit-int(count))))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(ttl).Unix(), 10))

		if int(count) > rule.Limit {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			response.Fail(c, 429, response.CodeRateLimited,
				"terlalu banyak permintaan, coba lagi nanti", map[string]any{
					"limit":       rule.Limit,
					"window":      rule.Window.String(),
					"retry_after": retryAfter,
				})
			return
		}
		c.Next()
	}
}
