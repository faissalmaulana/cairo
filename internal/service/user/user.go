package user_service

import (
	"context"
	"errors"

	"github.com/faissalmaulana/cairo/internal/model"
	"github.com/faissalmaulana/cairo/internal/repository/user"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo user_repository.UserRepository
}

func NewUserService(repo user_repository.UserRepository) *UserService {
	return &UserService{
		repo: repo,
	}
}

type SignUpInput struct {
	Username string
	Email    string
	Password string
}

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
)

func (us *UserService) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	user, err := us.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (us *UserService) GetByID(ctx context.Context, id string) (*model.User, error) {
	user, err := us.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (us *UserService) Create(ctx context.Context, nu SignUpInput) (string, error) {
	newUser, err := PrepareSignUp(nu)
	if err != nil {
		return "", err
	}

	id, err := us.repo.Create(ctx, *newUser)
	if err != nil {
		return "", errors.New("can't create new user")
	}

	return id, nil
}

// PrepareSignUp hashes the password and builds the user model ready for
// persistence. Extracted so callers that persist inside a transaction can
// reuse the same preparation without duplicating hashing logic.
func PrepareSignUp(nu SignUpInput) (*model.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(nu.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return &model.User{
		Username: nu.Username,
		Email:    nu.Email,
		Password: string(hashedPassword),
	}, nil
}

func (us *UserService) VerifyPassword(ctx context.Context, email, password string) (bool, error) {
	pass, err := us.repo.GetPasswordByEmail(ctx, email)
	if err != nil {
		return false, errors.New("can't find password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(pass), []byte(password)); err != nil {
		return false, errors.New("password mismatch")
	}

	return true, nil
}

// Authenticate returns the user when the email/password pair is valid.
func (us *UserService) Authenticate(ctx context.Context, email, password string) (*model.User, error) {
	user, err := us.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	pass, err := us.repo.GetPasswordByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(pass), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

func (us *UserService) EmailExists(ctx context.Context, email string) (bool, error) {

	isExist, err := us.repo.EmailExists(ctx, email)
	if err != nil {
		return false, errors.New("something went wrong")
	}

	return isExist, nil
}
