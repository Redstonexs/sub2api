package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	// These keys are intentionally separate from the general settings document.
	SettingKeyAntigravityOAuthClientID               = "antigravity_oauth_client_id"
	SettingKeyAntigravityOAuthClientSecretCiphertext = "antigravity_oauth_client_secret_ciphertext"

	antigravityOAuthSecretEnvelopeVersion = "v1:"
)

const (
	AntigravityOAuthCredentialSourceSettings    = "settings"
	AntigravityOAuthCredentialSourceEnvironment = "environment"
	AntigravityOAuthCredentialSourceNone        = "none"
)

var (
	ErrAntigravityOAuthCredentialsInvalid = infraerrors.BadRequest(
		"ANTIGRAVITY_OAUTH_CLIENT_CONFIGURATION_INVALID",
		"stored antigravity oauth client configuration is invalid",
	)
	ErrAntigravityOAuthCredentialsRequired = infraerrors.BadRequest(
		"ANTIGRAVITY_OAUTH_CLIENT_CONFIGURATION_REQUIRED",
		"antigravity oauth client_id and client_secret are required",
	)
)

// AntigravityOAuthCredentialStatus is safe to expose to an administrative
// handler. It deliberately never contains the client secret.
type AntigravityOAuthCredentialStatus struct {
	ClientID                string `json:"client_id"`
	ClientSecretConfigured  bool   `json:"client_secret_configured"`
	Source                  string `json:"source"`
	Valid                   bool   `json:"valid"`
	EncryptionKeyConfigured bool   `json:"encryption_key_configured"`
}

// UpdateAntigravityOAuthCredentialsInput is the admin mutation contract. A nil
// or empty ClientSecret means preserve the existing database secret.
type UpdateAntigravityOAuthCredentialsInput struct {
	ClientID     string
	ClientSecret *string
}

// SetSecretEncryptor injects the existing TOTP-backed secret encryptor. The
// optional flag is useful to isolated tests; production derives the flag from
// the fixed TOTP_ENCRYPTION_KEY configuration.
func (s *SettingService) SetSecretEncryptor(encryptor SecretEncryptor, configured ...bool) {
	if s == nil {
		return
	}
	keyConfigured := s.cfg != nil && s.cfg.Totp.EncryptionKeyConfigured
	if len(configured) > 0 {
		keyConfigured = configured[0]
	}
	s.antigravityOAuthCredentialMu.Lock()
	s.antigravityOAuthCredentialEncryptorValue = encryptor
	s.antigravityOAuthEncryptionKeyConfiguredValue = keyConfigured
	s.antigravityOAuthCredentialStateSet = true
	s.antigravityOAuthCredentialMu.Unlock()
}

// SetAntigravityOAuthEncryptor is a descriptive alias for handler/provider
// wiring and keeps the generic setter available to existing service tests.
func (s *SettingService) SetAntigravityOAuthEncryptor(encryptor SecretEncryptor, configured ...bool) {
	s.SetSecretEncryptor(encryptor, configured...)
}

func (s *SettingService) antigravityOAuthEncryptionKeyConfigured() bool {
	if s == nil {
		return false
	}
	s.antigravityOAuthCredentialMu.RLock()
	configured := s.antigravityOAuthEncryptionKeyConfiguredValue
	stateSet := s.antigravityOAuthCredentialStateSet
	s.antigravityOAuthCredentialMu.RUnlock()
	if stateSet {
		return configured
	}
	return s.cfg != nil && s.cfg.Totp.EncryptionKeyConfigured
}

func (s *SettingService) readAntigravityOAuthCredentialSettings(ctx context.Context) (map[string]string, error) {
	if s == nil || s.settingRepo == nil {
		return nil, errors.New("antigravity oauth settings repository is not configured")
	}
	values, err := s.settingRepo.GetMultiple(ctx, []string{
		SettingKeyAntigravityOAuthClientID,
		SettingKeyAntigravityOAuthClientSecretCiphertext,
	})
	if err != nil {
		return nil, fmt.Errorf("read antigravity oauth credentials: %w", err)
	}
	return values, nil
}

func antigravityOAuthDatabaseOverride(values map[string]string) bool {
	return strings.TrimSpace(values[SettingKeyAntigravityOAuthClientID]) != "" ||
		strings.TrimSpace(values[SettingKeyAntigravityOAuthClientSecretCiphertext]) != ""
}

func decryptAntigravityOAuthSecret(encryptor SecretEncryptor, envelope string) (string, error) {
	envelope = strings.TrimSpace(envelope)
	if !strings.HasPrefix(envelope, antigravityOAuthSecretEnvelopeVersion) {
		return "", ErrAntigravityOAuthCredentialsInvalid
	}
	if encryptor == nil {
		return "", ErrAntigravityOAuthCredentialsInvalid
	}
	secret, err := encryptor.Decrypt(strings.TrimPrefix(envelope, antigravityOAuthSecretEnvelopeVersion))
	if err != nil || strings.TrimSpace(secret) == "" {
		return "", ErrAntigravityOAuthCredentialsInvalid
	}
	return strings.TrimSpace(secret), nil
}

func (s *SettingService) resolveAntigravityOAuthCredentialPair(ctx context.Context) (antigravity.OAuthClientCredentials, string, error) {
	values, err := s.readAntigravityOAuthCredentialSettings(ctx)
	if err != nil {
		return antigravity.OAuthClientCredentials{}, "", err
	}

	if antigravityOAuthDatabaseOverride(values) {
		clientID := strings.TrimSpace(values[SettingKeyAntigravityOAuthClientID])
		clientSecret, decryptErr := decryptAntigravityOAuthSecret(
			s.antigravityOAuthCredentialEncryptor(),
			values[SettingKeyAntigravityOAuthClientSecretCiphertext],
		)
		if clientID == "" || decryptErr != nil {
			return antigravity.OAuthClientCredentials{}, AntigravityOAuthCredentialSourceSettings, ErrAntigravityOAuthCredentialsInvalid
		}
		return antigravity.OAuthClientCredentials{ClientID: clientID, ClientSecret: clientSecret}, AntigravityOAuthCredentialSourceSettings, nil
	}

	credentials, err := antigravity.ResolveOAuthClientCredentialsFromEnv()
	if err != nil {
		return antigravity.OAuthClientCredentials{}, AntigravityOAuthCredentialSourceEnvironment, err
	}
	return credentials, AntigravityOAuthCredentialSourceEnvironment, nil
}

func (s *SettingService) antigravityOAuthCredentialEncryptor() SecretEncryptor {
	if s == nil {
		return nil
	}
	s.antigravityOAuthCredentialMu.RLock()
	encryptor := s.antigravityOAuthCredentialEncryptorValue
	s.antigravityOAuthCredentialMu.RUnlock()
	return encryptor
}

// ResolveAntigravityOAuthClientCredentials atomically selects either the
// complete database pair or the complete environment pair. It never mixes
// sources and fails closed for any non-empty database override that is not a
// valid complete pair.
func (s *SettingService) ResolveAntigravityOAuthClientCredentials(ctx context.Context) (antigravity.OAuthClientCredentials, error) {
	credentials, _, err := s.resolveAntigravityOAuthCredentialPair(ctx)
	return credentials, err
}

// ResolveAntigravityOAuthCredentials is a shorter compatibility alias for the
// handler/service lane.
func (s *SettingService) ResolveAntigravityOAuthCredentials(ctx context.Context) (antigravity.OAuthClientCredentials, error) {
	return s.ResolveAntigravityOAuthClientCredentials(ctx)
}

// GetAntigravityOAuthCredentialStatus returns only non-secret credential
// metadata. A malformed database override is reported as invalid rather than
// being silently replaced by environment values.
func (s *SettingService) GetAntigravityOAuthCredentialStatus(ctx context.Context) (*AntigravityOAuthCredentialStatus, error) {
	values, err := s.readAntigravityOAuthCredentialSettings(ctx)
	if err != nil {
		return nil, err
	}
	status := &AntigravityOAuthCredentialStatus{
		EncryptionKeyConfigured: s.antigravityOAuthEncryptionKeyConfigured(),
	}
	if antigravityOAuthDatabaseOverride(values) {
		status.Source = AntigravityOAuthCredentialSourceSettings
		status.ClientID = strings.TrimSpace(values[SettingKeyAntigravityOAuthClientID])
		status.ClientSecretConfigured = strings.TrimSpace(values[SettingKeyAntigravityOAuthClientSecretCiphertext]) != ""
		if status.ClientID != "" {
			_, err := decryptAntigravityOAuthSecret(s.antigravityOAuthCredentialEncryptor(), values[SettingKeyAntigravityOAuthClientSecretCiphertext])
			status.Valid = err == nil
		}
		return status, nil
	}

	status.Source = AntigravityOAuthCredentialSourceEnvironment
	status.ClientID = strings.TrimSpace(getenv(antigravity.AntigravityOAuthClientIDEnv))
	status.ClientSecretConfigured = strings.TrimSpace(getenv(antigravity.AntigravityOAuthClientSecretEnv)) != ""
	status.Valid = status.ClientID != "" && status.ClientSecretConfigured
	if !status.Valid && status.ClientID == "" && !status.ClientSecretConfigured {
		status.Source = AntigravityOAuthCredentialSourceNone
	}
	return status, nil
}

// getenv is kept as a variable-free helper so credential status cannot expose
// the environment secret itself.
func getenv(key string) string {
	return os.Getenv(key)
}

// UpdateAntigravityOAuthCredentials validates and atomically persists the
// complete pair. Notification happens only after SetMultiple commits.
func (s *SettingService) UpdateAntigravityOAuthCredentials(ctx context.Context, input UpdateAntigravityOAuthCredentialsInput) error {
	if !s.antigravityOAuthEncryptionKeyConfigured() || s.antigravityOAuthCredentialEncryptor() == nil {
		return ErrSecretEncryptionKeyNotConfigured
	}
	clientID := strings.TrimSpace(input.ClientID)
	if clientID == "" {
		return ErrAntigravityOAuthCredentialsRequired
	}

	values, err := s.readAntigravityOAuthCredentialSettings(ctx)
	if err != nil {
		return err
	}
	databaseOverride := antigravityOAuthDatabaseOverride(values)
	secret := ""
	encryptedEnvelope := ""
	newSecret := input.ClientSecret != nil && strings.TrimSpace(*input.ClientSecret) != ""
	if newSecret {
		secret = strings.TrimSpace(*input.ClientSecret)
	} else {
		if !databaseOverride {
			return ErrAntigravityOAuthCredentialsRequired
		}
		storedID := strings.TrimSpace(values[SettingKeyAntigravityOAuthClientID])
		storedSecret, decryptErr := decryptAntigravityOAuthSecret(s.antigravityOAuthCredentialEncryptor(), values[SettingKeyAntigravityOAuthClientSecretCiphertext])
		if storedID == "" || decryptErr != nil {
			return ErrAntigravityOAuthCredentialsInvalid
		}
		if storedID != clientID {
			return ErrAntigravityOAuthCredentialsRequired
		}
		// Preserve the existing envelope verbatim when the secret was omitted;
		// this avoids needlessly rotating ciphertext for an ID-only edit.
		secret = storedSecret
		encryptedEnvelope = strings.TrimSpace(values[SettingKeyAntigravityOAuthClientSecretCiphertext])
	}

	if encryptedEnvelope == "" {
		encrypted, encryptErr := s.antigravityOAuthCredentialEncryptor().Encrypt(secret)
		if encryptErr != nil || strings.TrimSpace(encrypted) == "" {
			if encryptErr == nil {
				encryptErr = errors.New("empty encrypted antigravity oauth client secret")
			}
			return fmt.Errorf("encrypt antigravity oauth client secret: %w", encryptErr)
		}
		encryptedEnvelope = antigravityOAuthSecretEnvelopeVersion + strings.TrimSpace(encrypted)
	}
	updates := map[string]string{
		SettingKeyAntigravityOAuthClientID:               clientID,
		SettingKeyAntigravityOAuthClientSecretCiphertext: encryptedEnvelope,
	}
	if err := s.settingRepo.SetMultiple(ctx, updates); err != nil {
		return err
	}
	s.NotifySettingsChanged()
	return nil
}

// ClearAntigravityOAuthCredentials removes the database override atomically;
// empty values intentionally restore the environment fallback.
func (s *SettingService) ClearAntigravityOAuthCredentials(ctx context.Context) error {
	if s == nil || s.settingRepo == nil {
		return errors.New("antigravity oauth settings repository is not configured")
	}
	if err := s.settingRepo.SetMultiple(ctx, map[string]string{
		SettingKeyAntigravityOAuthClientID:               "",
		SettingKeyAntigravityOAuthClientSecretCiphertext: "",
	}); err != nil {
		return err
	}
	s.NotifySettingsChanged()
	return nil
}
