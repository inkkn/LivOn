package twilio

import (
	"context"
	"encoding/json"
	"fmt"
	"livon/internal/config"
	"net/http"
	"net/url"
	"strings"
)

type TwilioClient struct {
	SID       string
	Token     string
	VerifySID string
}

func NewTwilioClient(
	cfg config.TwilioConfig,
) *TwilioClient {
	return &TwilioClient{
		SID:       cfg.SID,
		Token:     cfg.Token,
		VerifySID: cfg.VerifySID,
	}
}

func (t *TwilioClient) SendOTP(ctx context.Context, phone string) error {
	apiURL := fmt.Sprintf("https://verify.twilio.com/v2/Services/%s/Verifications", t.VerifySID)

	data := url.Values{}
	data.Set("To", phone)
	data.Set("Channel", "sms")

	req, _ := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data.Encode()))
	req.SetBasicAuth(t.SID, t.Token)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode >= 400 {
		return fmt.Errorf("twilio error: %v", err)
	}
	return nil
}

func (t *TwilioClient) VerifyOTP(ctx context.Context, phone, code string) (bool, error) {
	apiURL := fmt.Sprintf("https://verify.twilio.com/v2/Services/%s/VerificationCheck", t.VerifySID)

	data := url.Values{}
	data.Set("To", phone)
	data.Set("Code", code)

	req, _ := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data.Encode()))
	req.SetBasicAuth(t.SID, t.Token)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}

	var result struct {
		Status string `json:"status"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	return result.Status == "approved", nil
}
