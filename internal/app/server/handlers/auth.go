package handlers

import (
	"encoding/json"
	"livon/internal/core/services"
	"livon/pkg/middleware"
	"log/slog"
	"net/http"
)

type AuthHandler struct {
	userSvc  *services.UserService
	tokenSvc *services.TokenService
}

func NewAuthHandler(u *services.UserService, t *services.TokenService) *AuthHandler {
	return &AuthHandler{userSvc: u, tokenSvc: t}
}

// Requesting the OTP
func (h *AuthHandler) RequestOTP(w http.ResponseWriter, r *http.Request) {
	log, _ := r.Context().Value(middleware.LoggerKey).(*slog.Logger)
	var req struct {
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.ErrorContext(r.Context(), "auth handler - bad request")
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if err := h.userSvc.RequestOTP(r.Context(), req.Phone); err != nil {
		log.ErrorContext(r.Context(), "auth handler - request otp failed", "phone", req.Phone)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "OTP sent successfully"})
	log.InfoContext(r.Context(), "auth handler - request otp sent", "phone", req.Phone)
}

// Verifying and Creating the Identity
func (h *AuthHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	log, _ := r.Context().Value(middleware.LoggerKey).(*slog.Logger)
	var req struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.ErrorContext(r.Context(), "auth handler - bad request")
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	// Verify OTP and Create/Get User in DB
	user, err := h.userSvc.VerifyOTP(r.Context(), req.Phone, req.Code)
	if err != nil {
		log.ErrorContext(r.Context(), "auth handler - verify otp failed", "phone", req.Phone, "code", req.Code)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	log.InfoContext(r.Context(), "auth handler - verify otp success", "phone", req.Phone, "code", req.Code)
	// Generate the JWT using the phone number as 'sub'
	token, err := h.tokenSvc.GenerateToken(user.ID) // user.ID is the phone number
	if err != nil {
		log.ErrorContext(r.Context(), "auth handler - generate token failed", "phone", req.Phone)
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}
	// Return Response
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":      token,
		"user_id":    user.ID,
		"created_at": user.CreatedAt,
	})
	log.InfoContext(r.Context(), "auth handler - token send success", "phone", req.Phone)
}
