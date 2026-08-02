package middleware

import (
	"github.com/faissalmaulana/cairo/internal/handler"
	"github.com/faissalmaulana/cairo/internal/service/user"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type AuthMiddleware struct {
	userService *user_service.UserService
}

func NewAuthMiddleware(userService *user_service.UserService) *AuthMiddleware {
	return &AuthMiddleware{
		userService: userService,
	}
}

func (am *AuthMiddleware) CheckAuth(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")
	if user == nil {
		handler.FailError(c, handler.ErrRequiredSignIn)
		c.Abort()
		return
	}

	usrEmail := user.(string)

	if isexist, err := am.userService.EmailExists(c.Request.Context(), usrEmail); err != nil || !isexist {
		if err != nil {
			handler.FailError(c, handler.ErrInternalServer)
			c.Abort()
			return
		}

		handler.FailError(c, handler.ErrRequiredSignIn)
		c.Abort()
		return
	}

	c.Set("auth_user", usrEmail)

	c.Next()
}
