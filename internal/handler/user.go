package handler

import (
	"net/http"

	"github.com/faissalmaulana/cairo/internal/service/user"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService *user_service.UserService
}

func NewUserHandler(userService *user_service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

type SignUpRequest struct {
	Username string `json:"username" binding:"required,min=3,max=30"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

func (ur *UserHandler) SignUp(c *gin.Context) {
	var signUp SignUpRequest
	if err := c.ShouldBindJSON(&signUp); err != nil {
		FailError(c, ErrValidation(err))
		return
	}

	exists, err := ur.userService.EmailExists(c.Request.Context(), signUp.Email)
	if err != nil {
		FailError(c, ErrInternalServer)
		return
	}
	if exists {
		FailError(c, ErrEmailAlreadyExists)
		return
	}

	if err := ur.userService.Create(c.Request.Context(), user_service.SignUpInput(signUp)); err != nil {
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
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (ur *UserHandler) SignIn(c *gin.Context) {
	var signIn SignInRequest

	if err := c.ShouldBindJSON(&signIn); err != nil {
		FailError(c, ErrValidation(err))
		return
	}

	isVerified, err := ur.userService.VerifyPassword(c.Request.Context(), signIn.Email, signIn.Password)
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
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

func (ur *UserHandler) Account(c *gin.Context) {
	authenticatedUserEmail := c.GetString("auth_user")

	user, err := ur.userService.GetByEmail(c.Request.Context(), authenticatedUserEmail)
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

func (ur *UserHandler) Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Options(sessions.Options{Path: "/", MaxAge: -1})

	if err := session.Save(); err != nil {
		FailError(c, ErrLogoutFailed)
		return
	}

	OK(c, http.StatusOK, gin.H{"message": "logged out"})
}
