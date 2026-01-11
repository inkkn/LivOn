package services

import (
	"context"
	"errors"
	"fmt"
	"livon/internal/core/contracts" // wherever your interfaces live
	"livon/internal/core/domain"
)

type UserService struct {
	repo   domain.UserRepository
	twilio contracts.Twilio
}

func NewUserService(repo domain.UserRepository, twilio contracts.Twilio) *UserService {
	return &UserService{
		repo:   repo,
		twilio: twilio,
	}
}

// RequestOTP initiates the registration/login process
func (s *UserService) RequestOTP(ctx context.Context, phone string) error {
	if phone == "" {
		return errors.New("phone number is required")
	}
	return s.twilio.SendOTP(ctx, phone)
}

// VerifyOTP checks the code and handles the user lifecycle
func (s *UserService) VerifyOTP(ctx context.Context, phone, code string) (*domain.User, error) {
	// Verify with Twilio
	isValid, err := s.twilio.VerifyOTP(ctx, phone, code)
	if err != nil {
		return nil, fmt.Errorf("verification service error: %w", err)
	}
	if !isValid {
		return nil, errors.New("invalid or expired OTP")
	}

	// Persist user (CreateUser uses ON CONFLICT, so it handles existing users)
	user, err := s.repo.CreateUser(ctx, phone)
	if err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	return user, nil
}
