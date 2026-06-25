package service

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveSMTPSecurity(t *testing.T) {
	tests := []struct {
		name   string
		port   int
		useTLS bool
		want   smtpSecurity
	}{
		{"465 always implicit TLS even when UseTLS off", 465, false, smtpSecurityImplicitTLS},
		{"465 implicit TLS with UseTLS on", 465, true, smtpSecurityImplicitTLS},
		{"587 always STARTTLS even when UseTLS on", 587, true, smtpSecuritySTARTTLS},
		{"587 STARTTLS with UseTLS off", 587, false, smtpSecuritySTARTTLS},
		{"25 always STARTTLS", 25, true, smtpSecuritySTARTTLS},
		{"2525 always STARTTLS", 2525, true, smtpSecuritySTARTTLS},
		{"non-standard port honors UseTLS on", 1025, true, smtpSecurityImplicitTLS},
		{"non-standard port honors UseTLS off", 1025, false, smtpSecuritySTARTTLS},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveSMTPSecurity(&SMTPConfig{Port: tt.port, UseTLS: tt.useTLS})
			require.Equal(t, tt.want, got, "port=%d useTLS=%v", tt.port, tt.useTLS)
		})
	}
}

func TestSMTPAuthNilWhenNoUsername(t *testing.T) {
	require.Nil(t, smtpAuth(&SMTPConfig{Username: "   "}), "blank username must yield no auth (unauthenticated relay)")
	require.NotNil(t, smtpAuth(&SMTPConfig{Username: "user", Host: "smtp.example.com"}))
}

type fakeTimeoutErr struct{}

func (fakeTimeoutErr) Error() string   { return "dial tcp 1.2.3.4:587: i/o timeout" }
func (fakeTimeoutErr) Timeout() bool   { return true }
func (fakeTimeoutErr) Temporary() bool { return true }

func TestSMTPDialErrorTimeoutHint(t *testing.T) {
	err := smtpDialError("smtp.example.com:587", smtpSecuritySTARTTLS, fakeTimeoutErr{})
	require.ErrorContains(t, err, "blocking outbound SMTP")
	require.ErrorContains(t, err, "DigitalOcean")
	require.ErrorContains(t, err, "STARTTLS")

	// Non-timeout errors keep the plain wrapper without the firewall guidance.
	plain := smtpDialError("smtp.example.com:587", smtpSecurityImplicitTLS, errors.New("connection refused"))
	require.ErrorContains(t, plain, "connect to smtp.example.com:587")
	require.NotContains(t, plain.Error(), "DigitalOcean")
}
