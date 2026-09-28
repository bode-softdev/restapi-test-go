package middleware

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"

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
		actor, err := auth.ValidateAccessToken(c.Request.Context(), raw)
		if err != nil {
			response.Fail(c, 401, tokenErrorCode(err), err.Error(), nil)
			return
		}
		c.Set(ctxUserID, actor.UserID)
		c.Set(ctxUsername, actor.Username)
		c.Set(ctxRole, actor.Role)
		c.Next()
	}
}

func tokenErrorCode(err error) string {
	switch {
	case errors.Is(err, service.ErrTokenExpired):
		return response.CodeTokenExpired
	case errors.Is(err, service.ErrTokenRevoked):
		return response.CodeTokenRevoked
	default:
		return response.CodeTokenInvalid
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
	parts := strings.SplitN(c.GetHeader("Authorization"), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	token := strings.TrimSpace(parts[1])
	return token, token != ""
}
