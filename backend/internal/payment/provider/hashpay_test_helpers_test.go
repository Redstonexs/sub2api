//go:build unit

package provider

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func newHashPayTestKey(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)
	return privateKey, string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}

func verifyHashPayRequestSignature(t *testing.T, publicKey *rsa.PublicKey, method, path, timestamp string, body []byte, encodedSignature string) {
	t.Helper()

	signature, err := base64.StdEncoding.DecodeString(encodedSignature)
	require.NoError(t, err)
	payload := strings.Join([]string{method, path, timestamp, string(body)}, "\n")
	digest := sha256.Sum256([]byte(payload))
	require.NoError(t, rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, digest[:], signature))
}

func encryptHashPayCallback(t *testing.T, publicKey *rsa.PublicKey, payload map[string]any) string {
	t.Helper()

	plaintext, err := json.Marshal(payload)
	require.NoError(t, err)
	contentKey := make([]byte, 32)
	_, err = rand.Read(contentKey)
	require.NoError(t, err)
	iv := make([]byte, 12)
	_, err = rand.Read(iv)
	require.NoError(t, err)

	block, err := aes.NewCipher(contentKey)
	require.NoError(t, err)
	gcm, err := cipher.NewGCM(block)
	require.NoError(t, err)
	encryptedKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, publicKey, contentKey, nil)
	require.NoError(t, err)

	body, err := json.Marshal(struct {
		Alg  string `json:"alg"`
		Data string `json:"data"`
		IV   string `json:"iv"`
		Key  string `json:"key"`
	}{
		Alg:  "RSA-OAEP-256+A256GCM",
		Data: base64.StdEncoding.EncodeToString(gcm.Seal(nil, iv, plaintext, nil)),
		IV:   base64.StdEncoding.EncodeToString(iv),
		Key:  base64.StdEncoding.EncodeToString(encryptedKey),
	})
	require.NoError(t, err)
	return string(body)
}
