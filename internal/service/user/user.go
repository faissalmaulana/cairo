package user_service

import (
	"context"
	"errors"

	"github.com/faissalmaulana/cairo/internal/model"
	userRepository "github.com/faissalmaulana/cairo/internal/repository/user"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	UsrRepo userRepository.UserRepository
}

func NewUserService(usrrepo userRepository.UserRepository) *UserService {
	return &UserService{
		UsrRepo: usrrepo,
	}
}

type NewUser struct {
	Username string
	Email    string
	Password string
}

func (us *UserService) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	user, err := us.UsrRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (us *UserService) Create(ctx context.Context, nu NewUser) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(nu.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	newUser := model.User{
		Username: nu.Username,
		Email:    nu.Email,
		Password: string(hashedPassword),
	}

	if err := us.UsrRepo.Create(ctx, newUser); err != nil {
		return errors.New("can't create new user")
	}

	return nil
}

func (us *UserService) VerifyPassword(ctx context.Context, email, password string) (bool, error) {
	pass, err := us.UsrRepo.GetPasswordByEmail(ctx, email)
	if err != nil {
		return false, errors.New("can't find password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(pass), []byte(password)); err != nil {
		return false, errors.New("password mismatch")
	}

	return true, nil
}

func (us *UserService) CheckUserByEmail(ctx context.Context, email string) (bool, error) {

	isExist, err := us.UsrRepo.FindUserByEmail(ctx, email)
	if err != nil {
		return false, errors.New("something went wrong")
	}

	return isExist, nil
}
