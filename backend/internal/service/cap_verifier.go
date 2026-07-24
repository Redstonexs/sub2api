package service

import "context"

type CapVerifier interface {
	VerifyToken(ctx context.Context, apiEndpoint, siteKey, secretKey, token string) (*CapVerifyResponse, error)
}

type CapVerifyResponse struct {
	Success bool `json:"success"`
}
