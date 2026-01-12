package domain

import "errors"

var (
	ErrInvalidConversationID     = errors.New("invalid conversation id")
	ErrConversationNotFound      = errors.New("conversation not found")
	ErrConversationAlreadyExists = errors.New("conversation already exists")
	ErrSequenceNotInitialized    = errors.New("conversation sequence not initialized")
	ErrInvalidParticipantID      = errors.New("invalid participant id")
	ErrParticipantNotFound       = errors.New("participant not found")
	ErrInvalidUserID             = errors.New("invalid user id")
	ErrUserNotFound              = errors.New("user not found")
)
