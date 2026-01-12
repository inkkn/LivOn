package services

import (
	"context"
	"errors"
	"fmt"
	"livon/internal/core/contracts" // wherever your interfaces live
	"livon/internal/core/domain"
	"log/slog"
)

type UserService struct {
	log    *slog.Logger
	repo   domain.UserRepository
	twilio contracts.Twilio
}

func NewUserService(log *slog.Logger, repo domain.UserRepository, twilio contracts.Twilio) *UserService {
	return &UserService{
		log:    log,
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
		s.log.ErrorContext(ctx, "user - verify otp error", "error", err)
		return nil, fmt.Errorf("verification service error: %w", err)
	}
	if !isValid {
		s.log.ErrorContext(ctx, "user - invalid or expired OTP", "phone", phone)
		return nil, errors.New("invalid or expired OTP")
	}
	// Persist user (CreateUser uses ON CONFLICT, so it handles existing users)
	user, err := s.repo.CreateUser(ctx, phone)
	if err != nil {
		s.log.ErrorContext(ctx, "user - create user error", "phone", phone, "error", err)
		return nil, fmt.Errorf("failed to save user: %w", err)
	}
	return user, nil
}
