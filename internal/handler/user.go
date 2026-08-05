package handler

import (
	"net/http"
	"time"

	"github.com/faissalmaulana/cairo/internal/helpers"
	auth_service "github.com/faissalmaulana/cairo/internal/service/auth"
	token_service "github.com/faissalmaulana/cairo/internal/service/token"
	user_service "github.com/faissalmaulana/cairo/internal/service/user"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService  *user_service.UserService
	tokenService *token_service.TokenService
	authService  *auth_service.AuthService
}

func NewUserHandler(userService *user_service.UserService, tokenService *token_service.TokenService, authService *auth_service.AuthService) *UserHandler {
	return &UserHandler{
		userService:  userService,
		tokenService: tokenService,
		authService:  authService,
	}
}

type SignUpRequest struct {
	Username string `json:"username" binding:"required,min=3,max=30"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type TokenResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	ExpiresIn        int64  `json:"expires_in"`
	RefreshExpiresIn int64  `json:"refresh_expires_in"`
	APIKey           string `json:"api_key,omitempty"`
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

	userID, key, err := ur.authService.SignUp(c.Request.Context(), user_service.SignUpInput(signUp))
	if err != nil {
		FailError(c, ErrSignUpFailed)
		return
	}

	tokens, err := ur.issueTokens(c, userID)
	if err != nil {
		FailError(c, ErrSignUpFailed)
		return
	}

	tokens.APIKey = key.Plain

	OK(c, http.StatusCreated, tokens)
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

	user, err := ur.userService.Authenticate(c.Request.Context(), signIn.Email, signIn.Password)
	if err != nil {
		FailError(c, ErrInvalidCredentials)
		return
	}

	tokens, err := ur.issueTokens(c, user.ID)
	if err != nil {
		FailError(c, ErrInternalServer)
		return
	}

	OK(c, http.StatusOK, tokens)
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (ur *UserHandler) Refresh(c *gin.Context) {
	rawToken, err := helpers.BearerToken(c.GetHeader("Authorization"))
	if err != nil {
		FailError(c, ErrTokenRequired)
		return
	}

	claims, err := ur.tokenService.ParseRefreshToken(rawToken)
	if err != nil {
		FailError(c, ErrRefreshFailed)
		return
	}

	userID, err := ur.tokenService.ConsumeRefresh(c.Request.Context(), claims.ID)
	if err != nil {
		FailError(c, ErrRefreshFailed)
		return
	}

	tokens, err := ur.issueTokens(c, userID)
	if err != nil {
		FailError(c, ErrInternalServer)
		return
	}

	OK(c, http.StatusOK, tokens)
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (ur *UserHandler) Logout(c *gin.Context) {
	authUserID := c.GetString(helpers.AuthUserIDKey)
	if authUserID == "" {
		FailError(c, ErrTokenRequired)
		return
	}

	rawToken, err := helpers.BearerToken(c.GetHeader("Authorization"))
	if err != nil {
		FailError(c, ErrTokenRequired)
		return
	}

	accessClaims, err := ur.tokenService.ParseAccessToken(rawToken)
	if err != nil {
		FailError(c, ErrInvalidToken)
		return
	}

	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		FailError(c, ErrValidation(err))
		return
	}

	refreshClaims, err := ur.tokenService.ParseRefreshToken(req.RefreshToken)
	if err != nil {
		FailError(c, ErrInvalidToken)
		return
	}

	if err := ur.tokenService.Revoke(c.Request.Context(), accessClaims.ID, time.Until(accessClaims.ExpiresAt.Time)); err != nil {
		FailError(c, ErrLogoutFailed)
		return
	}

	if _, err := ur.tokenService.ConsumeRefresh(c.Request.Context(), refreshClaims.ID); err != nil {
		FailError(c, ErrLogoutFailed)
		return
	}

	OK(c, http.StatusOK, gin.H{"message": "logged out"})
}

type UserResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

func (ur *UserHandler) Account(c *gin.Context) {
	authUserID := c.GetString(helpers.AuthUserIDKey)

	user, err := ur.userService.GetByID(c.Request.Context(), authUserID)
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

func (ur *UserHandler) issueTokens(c *gin.Context, userID string) (*TokenResponse, error) {
	accessToken, _, err := ur.tokenService.GenerateAccessToken(userID)
	if err != nil {
		return nil, err
	}

	refreshToken, refreshClaims, err := ur.tokenService.GenerateRefreshToken(userID)
	if err != nil {
		return nil, err
	}

	if err := ur.tokenService.StoreRefresh(c.Request.Context(), refreshClaims.ID, userID); err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		TokenType:        "Bearer",
		ExpiresIn:        int64(ur.tokenService.AccessTTL().Seconds()),
		RefreshExpiresIn: int64(ur.tokenService.RefreshTTL().Seconds()),
	}, nil
}
