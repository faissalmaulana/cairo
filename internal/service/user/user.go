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

func (us *UserService) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	user, err := us.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (us *UserService) Create(ctx context.Context, nu SignUpInput) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(nu.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	newUser := model.User{
		Username: nu.Username,
		Email:    nu.Email,
		Password: string(hashedPassword),
	}

	if err := us.repo.Create(ctx, newUser); err != nil {
		return errors.New("can't create new user")
	}

	return nil
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

func (us *UserService) EmailExists(ctx context.Context, email string) (bool, error) {

	isExist, err := us.repo.EmailExists(ctx, email)
	if err != nil {
		return false, errors.New("something went wrong")
	}

	return isExist, nil
}
