//go:build unit

package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type antigravityCredentialsRepoStub struct {
	values      map[string]string
	setCalls    int
	lastUpdates map[string]string
	getErr      error
	setErr      error
}

func (r *antigravityCredentialsRepoStub) Get(context.Context, string) (*Setting, error) {
	return nil, ErrSettingNotFound
}
func (r *antigravityCredentialsRepoStub) GetValue(_ context.Context, key string) (string, error) {
	value, ok := r.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return value, nil
}
func (r *antigravityCredentialsRepoStub) Set(_ context.Context, key, value string) error {
	return r.SetMultiple(context.Background(), map[string]string{key: value})
}
func (r *antigravityCredentialsRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := r.values[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}
func (r *antigravityCredentialsRepoStub) SetMultiple(_ context.Context, values map[string]string) error {
	if r.setErr != nil {
		return r.setErr
	}
	r.setCalls++
	r.lastUpdates = make(map[string]string, len(values))
	for key, value := range values {
		r.values[key] = value
		r.lastUpdates[key] = value
	}
	return nil
}
func (r *antigravityCredentialsRepoStub) GetAll(context.Context) (map[string]string, error) {
	return r.values, nil
}
func (r *antigravityCredentialsRepoStub) Delete(_ context.Context, key string) error {
	delete(r.values, key)
	return nil
}

type antigravityCredentialsEncryptorStub struct{}

func (antigravityCredentialsEncryptorStub) Encrypt(value string) (string, error) {
	if value == "" {
		return "", errors.New("empty plaintext")
	}
	return "ciphertext-token", nil
}
func (antigravityCredentialsEncryptorStub) Decrypt(value string) (string, error) {
	if value != "ciphertext-token" {
		return "", errors.New("invalid ciphertext")
	}
	return "synthetic-secret", nil
}

type antigravityCredentialsEncryptorBStub struct{}

func (antigravityCredentialsEncryptorBStub) Encrypt(value string) (string, error) {
	if value == "" {
		return "", errors.New("empty plaintext")
	}
	return "ciphertext-token-b", nil
}

func (antigravityCredentialsEncryptorBStub) Decrypt(value string) (string, error) {
	if value != "ciphertext-token-b" {
		return "", errors.New("invalid ciphertext")
	}
	return "synthetic-secret-b", nil
}

func newAntigravityCredentialsSettingService(repo *antigravityCredentialsRepoStub, configured bool) *SettingService {
	svc := NewSettingService(repo, &config.Config{})
	svc.SetSecretEncryptor(antigravityCredentialsEncryptorStub{}, configured)
	return svc
}

func TestAntigravityOAuthCredentials_StoresVersionedCiphertext(t *testing.T) {
	repo := &antigravityCredentialsRepoStub{values: map[string]string{}}
	svc := newAntigravityCredentialsSettingService(repo, true)
	secret := "synthetic-secret"

	if err := svc.UpdateAntigravityOAuthCredentials(context.Background(), UpdateAntigravityOAuthCredentialsInput{
		ClientID:     " synthetic-client-id ",
		ClientSecret: &secret,
	}); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	stored := repo.values[SettingKeyAntigravityOAuthClientSecretCiphertext]
	if !strings.HasPrefix(stored, "v1:") || strings.Contains(stored, secret) {
		t.Fatalf("secret was not stored as versioned ciphertext: %q", stored)
	}
	credentials, err := svc.ResolveAntigravityOAuthClientCredentials(context.Background())
	if err != nil || credentials.ClientSecret != secret {
		t.Fatalf("resolved credentials = %+v, err=%v", credentials, err)
	}
}

func TestAntigravityOAuthCredentials_MissingKeyDoesNotMutate(t *testing.T) {
	repo := &antigravityCredentialsRepoStub{values: map[string]string{"existing": "value"}}
	svc := NewSettingService(repo, &config.Config{})
	secret := "synthetic-secret"

	err := svc.UpdateAntigravityOAuthCredentials(context.Background(), UpdateAntigravityOAuthCredentialsInput{
		ClientID:     "synthetic-client-id",
		ClientSecret: &secret,
	})
	if !errors.Is(err, ErrSecretEncryptionKeyNotConfigured) {
		t.Fatalf("error = %v, want encryption-key error", err)
	}
	if repo.setCalls != 0 || repo.values["existing"] != "value" {
		t.Fatalf("key failure mutated repository: calls=%d values=%v", repo.setCalls, repo.values)
	}
}

func TestAntigravityOAuthCredentials_DBPrecedenceAndFailClosed(t *testing.T) {
	t.Setenv("ANTIGRAVITY_OAUTH_CLIENT_ID", "synthetic-env-id")
	t.Setenv("ANTIGRAVITY_OAUTH_CLIENT_SECRET", "synthetic-env-secret")
	const dbID = "synthetic-db-id"
	repo := &antigravityCredentialsRepoStub{values: map[string]string{
		SettingKeyAntigravityOAuthClientID:               dbID,
		SettingKeyAntigravityOAuthClientSecretCiphertext: "v1:ciphertext-token",
	}}
	svc := newAntigravityCredentialsSettingService(repo, true)
	credentials, err := svc.ResolveAntigravityOAuthClientCredentials(context.Background())
	if err != nil || credentials.ClientID != dbID || credentials.ClientSecret != "synthetic-secret" {
		t.Fatalf("database credentials = %+v, err=%v", credentials, err)
	}
	status, err := svc.GetAntigravityOAuthCredentialStatus(context.Background())
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if status.Source != AntigravityOAuthCredentialSourceSettings || !status.Valid {
		t.Fatalf("status = %+v, want valid settings-backed status", status)
	}

	repo.values[SettingKeyAntigravityOAuthClientSecretCiphertext] = "plaintext-secret"
	if _, err := svc.ResolveAntigravityOAuthClientCredentials(context.Background()); err == nil {
		t.Fatal("malformed database secret should fail closed instead of using env")
	}
	status, err = svc.GetAntigravityOAuthCredentialStatus(context.Background())
	if err != nil {
		t.Fatalf("invalid status failed: %v", err)
	}
	if status.Source != AntigravityOAuthCredentialSourceSettings || status.Valid {
		t.Fatalf("invalid status = %+v, want settings and valid=false", status)
	}
}

func TestAntigravityOAuthCredentials_EnvironmentAndNoneStatus(t *testing.T) {
	repo := &antigravityCredentialsRepoStub{values: map[string]string{}}
	svc := newAntigravityCredentialsSettingService(repo, false)

	t.Setenv("ANTIGRAVITY_OAUTH_CLIENT_ID", "synthetic-env-id")
	t.Setenv("ANTIGRAVITY_OAUTH_CLIENT_SECRET", "synthetic-env-secret")
	status, err := svc.GetAntigravityOAuthCredentialStatus(context.Background())
	if err != nil {
		t.Fatalf("environment status failed: %v", err)
	}
	if status.Source != AntigravityOAuthCredentialSourceEnvironment || !status.Valid {
		t.Fatalf("environment status = %+v", status)
	}

	t.Setenv("ANTIGRAVITY_OAUTH_CLIENT_ID", "")
	t.Setenv("ANTIGRAVITY_OAUTH_CLIENT_SECRET", "")
	status, err = svc.GetAntigravityOAuthCredentialStatus(context.Background())
	if err != nil {
		t.Fatalf("none status failed: %v", err)
	}
	if status.Source != AntigravityOAuthCredentialSourceNone || status.Valid {
		t.Fatalf("none status = %+v", status)
	}
}

func TestAntigravityOAuthCredentials_StateIsPerService(t *testing.T) {
	repoA := &antigravityCredentialsRepoStub{values: map[string]string{
		SettingKeyAntigravityOAuthClientID:               "synthetic-client-id-a",
		SettingKeyAntigravityOAuthClientSecretCiphertext: "v1:ciphertext-token",
	}}
	repoB := &antigravityCredentialsRepoStub{values: map[string]string{
		SettingKeyAntigravityOAuthClientID:               "synthetic-client-id-b",
		SettingKeyAntigravityOAuthClientSecretCiphertext: "v1:ciphertext-token-b",
	}}
	svcA := NewSettingService(repoA, &config.Config{})
	svcB := NewSettingService(repoB, &config.Config{})
	svcA.SetSecretEncryptor(antigravityCredentialsEncryptorStub{}, true)
	svcB.SetSecretEncryptor(antigravityCredentialsEncryptorBStub{}, true)

	credentialsA, err := svcA.ResolveAntigravityOAuthClientCredentials(context.Background())
	if err != nil || credentialsA.ClientSecret != "synthetic-secret" {
		t.Fatalf("service A credentials = %+v, err=%v", credentialsA, err)
	}
	credentialsB, err := svcB.ResolveAntigravityOAuthClientCredentials(context.Background())
	if err != nil || credentialsB.ClientSecret != "synthetic-secret-b" {
		t.Fatalf("service B credentials = %+v, err=%v", credentialsB, err)
	}
}

func TestAntigravityOAuthCredentials_NotificationOnlyAfterCommit(t *testing.T) {
	repo := &antigravityCredentialsRepoStub{values: map[string]string{}}
	svc := newAntigravityCredentialsSettingService(repo, true)
	notifications := 0
	svc.SetOnUpdateCallback(func() { notifications++ })
	secret := "synthetic-secret"

	repo.setErr = errors.New("synthetic write failure")
	if err := svc.UpdateAntigravityOAuthCredentials(context.Background(), UpdateAntigravityOAuthCredentialsInput{ClientID: "synthetic-client-id", ClientSecret: &secret}); err == nil {
		t.Fatal("write failure should be returned")
	}
	if notifications != 0 {
		t.Fatalf("failed update notified %d times", notifications)
	}

	repo.setErr = nil
	if err := svc.UpdateAntigravityOAuthCredentials(context.Background(), UpdateAntigravityOAuthCredentialsInput{ClientID: "synthetic-client-id", ClientSecret: &secret}); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if notifications != 1 {
		t.Fatalf("successful update notified %d times", notifications)
	}
	if err := svc.ClearAntigravityOAuthCredentials(context.Background()); err != nil {
		t.Fatalf("clear failed: %v", err)
	}
	if notifications != 2 {
		t.Fatalf("successful clear notified %d times", notifications)
	}
}

func TestAntigravityOAuthCredentials_ClearWorksWithoutEncryptionKey(t *testing.T) {
	repo := &antigravityCredentialsRepoStub{values: map[string]string{
		SettingKeyAntigravityOAuthClientID:               "synthetic-client-id",
		SettingKeyAntigravityOAuthClientSecretCiphertext: "v1:ciphertext-token",
	}}
	svc := NewSettingService(repo, &config.Config{})
	notifications := 0
	svc.SetOnUpdateCallback(func() { notifications++ })

	if err := svc.ClearAntigravityOAuthCredentials(context.Background()); err != nil {
		t.Fatalf("keyless clear failed: %v", err)
	}
	if repo.setCalls != 1 || repo.values[SettingKeyAntigravityOAuthClientID] != "" ||
		repo.values[SettingKeyAntigravityOAuthClientSecretCiphertext] != "" {
		t.Fatalf("clear did not atomically clear credentials: calls=%d values=%v", repo.setCalls, repo.values)
	}
	if notifications != 1 {
		t.Fatalf("successful keyless clear notified %d times", notifications)
	}
}
