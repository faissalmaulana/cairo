package handler

import (
	"net/http"

	user_service "github.com/faissalmaulana/cairo/internal/service/user"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type UserResource struct {
	usrService *user_service.UserService
}

func NewUserResource(usrService *user_service.UserService) *UserResource {
	return &UserResource{
		usrService: usrService,
	}
}

type SignUpRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (ur *UserResource) SignUp(c *gin.Context) {
	var signUp SignUpRequest
	if err := c.ShouldBindJSON(&signUp); err != nil {
		FailError(c, ErrInvalidBodyRequest)
		return
	}

	if err := ur.usrService.Create(c.Request.Context(), user_service.NewUser(signUp)); err != nil {
		FailError(c, ErrSignUpFailed)
		return
	}

	session := sessions.Default(c)
	session.Set("user", signUp.Email)

	if err := session.Save(); err != nil {
		FailError(c, ErrSignUpFailed)
		return
	}

	OK(c, http.StatusCreated, gin.H{"message": "user successfully signed up"})
}

type SignInRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (ur *UserResource) SignIn(c *gin.Context) {
	var signIn SignInRequest

	if err := c.ShouldBindJSON(&signIn); err != nil {
		FailError(c, ErrInvalidBodyRequest)
		return
	}

	isVerified, err := ur.usrService.VerifyPassword(c.Request.Context(), signIn.Email, signIn.Password)
	if err != nil {
		FailError(c, ErrInvalidCredentials)
		return
	}

	if isVerified {
		session := sessions.Default(c)
		session.Set("user", signIn.Email)
		if err := session.Save(); err != nil {
			FailError(c, ErrInternalServer)
			return
		}
	}
	OK(c, http.StatusOK, gin.H{"message": "user successfully signed in"})
}

type UserResponse struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

func (ur *UserResource) Account(c *gin.Context) {
	authenticatedUserEmail := c.GetString("auth_user")

	user, err := ur.usrService.GetUserByEmail(c.Request.Context(), authenticatedUserEmail)
	if err != nil {
		FailError(c, ErrInternalServer)
		return
	}

	OK(c, http.StatusOK, UserResponse{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
	})
}

func (ur *UserResource) Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Options(sessions.Options{Path: "/", MaxAge: -1})

	if err := session.Save(); err != nil {
		FailError(c, ErrLogoutFailed)
		return
	}

	OK(c, http.StatusOK, gin.H{"message": "logged out"})
}
