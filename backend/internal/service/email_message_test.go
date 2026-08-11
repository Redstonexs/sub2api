//go:build unit

package service

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildSMTPMessageProducesStandardsCompliantMIME(t *testing.T) {
	config := &SMTPConfig{
		Host:     "smtp.example.com",
		From:     "reply@example.com",
		FromName: "Sub2API 通知",
	}
	body := "<html>\n<body>验证码：123456 &amp; ready</body>\n</html>"

	message, err := buildSMTPMessage(config, "User <user@example.net>", "邮箱验证码", body)
	require.NoError(t, err)
	require.Equal(t, "reply@example.com", message.envelopeFrom)
	require.Equal(t, "user@example.net", message.envelopeTo)

	parsed, err := mail.ReadMessage(bytes.NewReader(message.data))
	require.NoError(t, err)

	from, err := mail.ParseAddress(parsed.Header.Get("From"))
	require.NoError(t, err)
	require.Equal(t, "Sub2API 通知", from.Name)
	require.Equal(t, "reply@example.com", from.Address)

	recipient, err := mail.ParseAddress(parsed.Header.Get("To"))
	require.NoError(t, err)
	require.Equal(t, "User", recipient.Name)
	require.Equal(t, "user@example.net", recipient.Address)

	decodedSubject, err := new(mime.WordDecoder).DecodeHeader(parsed.Header.Get("Subject"))
	require.NoError(t, err)
	require.Equal(t, "邮箱验证码", decodedSubject)
	require.NotEmpty(t, parsed.Header.Get("Date"))
	_, err = mail.ParseDate(parsed.Header.Get("Date"))
	require.NoError(t, err)
	require.Regexp(t, regexp.MustCompile(`^<[0-9a-f]{32}@example\.com>$`), parsed.Header.Get("Message-ID"))
	require.Equal(t, "1.0", parsed.Header.Get("MIME-Version"))

	// An HTML body ships as multipart/alternative so clients that render the text
	// part have one, and so the message does not read as HTML-only to spam filters.
	mediaType, params, err := mime.ParseMediaType(parsed.Header.Get("Content-Type"))
	require.NoError(t, err)
	require.Equal(t, "multipart/alternative", mediaType)
	require.NotEmpty(t, params["boundary"])

	// multipart.Reader hides Content-Transfer-Encoding and decodes quoted-printable
	// transparently, so assert the raw bytes carry it.
	require.Contains(t, string(message.data), "Content-Transfer-Encoding: quoted-printable")

	reader := multipart.NewReader(parsed.Body, params["boundary"])

	// RFC 2046 §5.1.4: least-preferred alternative first, so text/plain precedes HTML.
	textPart, err := reader.NextPart()
	require.NoError(t, err)
	textType, textParams, err := mime.ParseMediaType(textPart.Header.Get("Content-Type"))
	require.NoError(t, err)
	require.Equal(t, "text/plain", textType)
	require.Equal(t, "UTF-8", textParams["charset"])
	textBody, err := io.ReadAll(textPart)
	require.NoError(t, err)
	require.Contains(t, string(textBody), "验证码：123456 & ready")
	require.NotContains(t, string(textBody), "<body")

	htmlPart, err := reader.NextPart()
	require.NoError(t, err)
	htmlType, htmlParams, err := mime.ParseMediaType(htmlPart.Header.Get("Content-Type"))
	require.NoError(t, err)
	require.Equal(t, "text/html", htmlType)
	require.Equal(t, "UTF-8", htmlParams["charset"])
	htmlBody, err := io.ReadAll(htmlPart)
	require.NoError(t, err)
	require.Equal(t, strings.ReplaceAll(body, "\n", "\r\n"), string(htmlBody))

	_, err = reader.NextPart()
	require.ErrorIs(t, err, io.EOF)
}

// A body with no readable text has nothing to alternate between, so it stays a
// single part rather than shipping an empty text/plain alternative.
func TestBuildSMTPMessageKeepsTextlessBodySinglePart(t *testing.T) {
	config := &SMTPConfig{Host: "smtp.example.com", From: "reply@example.com"}

	message, err := buildSMTPMessage(config, "user@example.net", "subject", `<html><body><style>p { color: red; }</style></body></html>`)
	require.NoError(t, err)

	parsed, err := mail.ReadMessage(bytes.NewReader(message.data))
	require.NoError(t, err)
	require.Equal(t, "quoted-printable", parsed.Header.Get("Content-Transfer-Encoding"))

	mediaType, params, err := mime.ParseMediaType(parsed.Header.Get("Content-Type"))
	require.NoError(t, err)
	require.Equal(t, "text/html", mediaType)
	require.Equal(t, "UTF-8", params["charset"])
}

// Each message gets its own boundary; a fixed one could collide with body content
// and truncate the message at the false delimiter.
func TestBuildSMTPMessageUsesUniqueBoundaries(t *testing.T) {
	config := &SMTPConfig{Host: "smtp.example.com", From: "reply@example.com"}

	boundaryOf := func(t *testing.T) string {
		t.Helper()
		message, err := buildSMTPMessage(config, "user@example.net", "subject", "<p>hello</p>")
		require.NoError(t, err)
		parsed, err := mail.ReadMessage(bytes.NewReader(message.data))
		require.NoError(t, err)
		_, params, err := mime.ParseMediaType(parsed.Header.Get("Content-Type"))
		require.NoError(t, err)
		return params["boundary"]
	}

	first, second := boundaryOf(t), boundaryOf(t)
	require.NotEmpty(t, first)
	require.NotEqual(t, first, second)
}

func TestBuildSMTPMessagePreventsHeaderInjection(t *testing.T) {
	config := &SMTPConfig{
		Host:     "smtp.example.com",
		From:     "reply@example.com",
		FromName: "Sender\r\nBcc: hidden@example.com",
	}

	message, err := buildSMTPMessage(config, "user@example.net", "Subject\r\nCc: hidden@example.com", "body")
	require.NoError(t, err)

	parsed, err := mail.ReadMessage(bytes.NewReader(message.data))
	require.NoError(t, err)
	require.Empty(t, parsed.Header.Get("Bcc"))
	require.Empty(t, parsed.Header.Get("Cc"))

	decodedSubject, err := new(mime.WordDecoder).DecodeHeader(parsed.Header.Get("Subject"))
	require.NoError(t, err)
	require.Equal(t, "SubjectCc: hidden@example.com", decodedSubject)
}

func TestBuildSMTPMessageRejectsInvalidConfiguration(t *testing.T) {
	_, err := buildSMTPMessage(nil, "user@example.net", "subject", "body")
	require.ErrorContains(t, err, "missing SMTP configuration")

	_, err = buildSMTPMessage(&SMTPConfig{Host: "smtp.example.com"}, "user@example.net", "subject", "body")
	require.ErrorContains(t, err, "invalid SMTP from address")

	_, err = buildSMTPMessage(&SMTPConfig{
		Host: "smtp.example.com",
		From: "reply@example.com",
	}, "invalid recipient <>", "subject", "body")
	require.ErrorContains(t, err, "invalid SMTP recipient address")

	_, err = buildSMTPMessage(&SMTPConfig{
		Host: "smtp.example.com",
		From: "reply@example.com",
	}, "user@example.net\r\nBcc: hidden@example.net", "subject", "body")
	require.ErrorContains(t, err, "invalid SMTP recipient address")
}

func TestBuildSMTPMessageUsesUniqueMessageIDs(t *testing.T) {
	config := &SMTPConfig{Host: "smtp.example.com", From: "reply@example.com"}

	first, err := buildSMTPMessage(config, "user@example.net", "subject", "body")
	require.NoError(t, err)
	second, err := buildSMTPMessage(config, "user@example.net", "subject", "body")
	require.NoError(t, err)

	firstParsed, err := mail.ReadMessage(bytes.NewReader(first.data))
	require.NoError(t, err)
	secondParsed, err := mail.ReadMessage(bytes.NewReader(second.data))
	require.NoError(t, err)
	require.NotEqual(t, firstParsed.Header.Get("Message-ID"), secondParsed.Header.Get("Message-ID"))
}
