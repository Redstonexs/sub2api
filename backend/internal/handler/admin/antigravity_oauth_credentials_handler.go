package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// antigravityOAuthCredentialStatusResponse is intentionally separate from the
// service type so this endpoint cannot accidentally expose a future secret
// field added to the service model.
type antigravityOAuthCredentialStatusResponse struct {
	ClientID                string `json:"client_id"`
	ClientSecretConfigured  bool   `json:"client_secret_configured"`
	Source                  string `json:"source"`
	Valid                   bool   `json:"valid"`
	EncryptionKeyConfigured bool   `json:"encryption_key_configured"`
}

func antigravityOAuthCredentialStatusDTO(status *service.AntigravityOAuthCredentialStatus) antigravityOAuthCredentialStatusResponse {
	if status == nil {
		return antigravityOAuthCredentialStatusResponse{}
	}
	return antigravityOAuthCredentialStatusResponse{
		ClientID:                status.ClientID,
		ClientSecretConfigured:  status.ClientSecretConfigured,
		Source:                  status.Source,
		Valid:                   status.Valid,
		EncryptionKeyConfigured: status.EncryptionKeyConfigured,
	}
}

// GetAntigravityOAuthCredentials returns only non-secret credential metadata.
// GET /api/v1/admin/settings/antigravity-oauth-credentials
func (h *SettingHandler) GetAntigravityOAuthCredentials(c *gin.Context) {
	status, err := h.settingService.GetAntigravityOAuthCredentialStatus(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, antigravityOAuthCredentialStatusDTO(status))
}

type updateAntigravityOAuthCredentialsRequest struct {
	ClientID     string  `json:"client_id"`
	ClientSecret *string `json:"client_secret"`
}

// UpdateAntigravityOAuthCredentials stores the credential pair and returns
// fresh non-secret metadata. A pointer preserves omitted versus explicitly
// supplied client_secret when handing the request to the service layer.
// PUT /api/v1/admin/settings/antigravity-oauth-credentials
func (h *SettingHandler) UpdateAntigravityOAuthCredentials(c *gin.Context) {
	var req updateAntigravityOAuthCredentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Do not reflect binding details: malformed input could contain a
		// credential value supplied by the caller.
		response.BadRequest(c, "invalid request body")
		return
	}

	if err := h.settingService.UpdateAntigravityOAuthCredentials(c.Request.Context(), service.UpdateAntigravityOAuthCredentialsInput{
		ClientID:     req.ClientID,
		ClientSecret: req.ClientSecret,
	}); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	status, err := h.settingService.GetAntigravityOAuthCredentialStatus(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, antigravityOAuthCredentialStatusDTO(status))
}

// DeleteAntigravityOAuthCredentials removes the database override and restores
// environment fallback. The response deliberately contains only a stable
// operation status.
// DELETE /api/v1/admin/settings/antigravity-oauth-credentials
func (h *SettingHandler) DeleteAntigravityOAuthCredentials(c *gin.Context) {
	if err := h.settingService.ClearAntigravityOAuthCredentials(c.Request.Context()); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"status": "cleared"})
}
