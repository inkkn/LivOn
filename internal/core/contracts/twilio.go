package contracts

import "context"

type Twilio interface {
	SendOTP(ctx context.Context, phone string) error
	VerifyOTP(ctx context.Context, phone, code string) (bool, error)
}
