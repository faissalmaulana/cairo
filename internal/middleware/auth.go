package middleware

import (
	"github.com/faissalmaulana/cairo/internal/handler"
	user_service "github.com/faissalmaulana/cairo/internal/service/user"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type AuthMiddleware struct {
	usrService *user_service.UserService
}

func NewAuthMiddleware(usrService *user_service.UserService) *AuthMiddleware {
	return &AuthMiddleware{
		usrService: usrService,
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

	if isexist, err := am.usrService.CheckUserByEmail(c.Request.Context(), usrEmail); err != nil || !isexist {
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
