package service

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"time"
)

type smtpMessage struct {
	envelopeFrom string
	envelopeTo   string
	data         []byte
}

func buildSMTPMessage(config *SMTPConfig, to, subject, body string) (smtpMessage, error) {
	if config == nil {
		return smtpMessage{}, errors.New("missing SMTP configuration")
	}

	fromAddress, err := parseSMTPAddress(config.From, "from")
	if err != nil {
		return smtpMessage{}, err
	}
	recipientAddress, err := parseSMTPAddress(to, "recipient")
	if err != nil {
		return smtpMessage{}, err
	}
	messageID, err := generateEmailMessageID(fromAddress.Address, config.Host)
	if err != nil {
		return smtpMessage{}, fmt.Errorf("generate message ID: %w", err)
	}

	fromName := sanitizeEmailHeader(config.FromName)
	if strings.TrimSpace(fromName) == "" {
		fromName = fromAddress.Name
	}
	fromHeader := (&mail.Address{
		Name:    fromName,
		Address: fromAddress.Address,
	}).String()
	toHeader := (&mail.Address{
		Name:    recipientAddress.Name,
		Address: recipientAddress.Address,
	}).String()
	subjectHeader := mime.QEncoding.Encode("UTF-8", sanitizeEmailHeader(subject))

	var message bytes.Buffer
	fmt.Fprintf(&message, "From: %s\r\n", fromHeader)
	fmt.Fprintf(&message, "To: %s\r\n", toHeader)
	fmt.Fprintf(&message, "Date: %s\r\n", time.Now().UTC().Format(time.RFC1123Z))
	fmt.Fprintf(&message, "Message-ID: %s\r\n", messageID)
	fmt.Fprintf(&message, "Subject: %s\r\n", subjectHeader)
	fmt.Fprint(&message, "MIME-Version: 1.0\r\n")

	// Pair the HTML with a text/plain alternative. HTML-only mail scores worse with
	// spam filters and leaves nothing for clients that render the text part.
	// htmlToPlainText returns "" for a body with no readable text, in which case
	// there is nothing to alternate between and a single part is correct.
	plain := htmlToPlainText(body)
	if plain == "" {
		fmt.Fprint(&message, "Content-Type: text/html; charset=UTF-8\r\n"+
			"Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		if err := writeQuotedPrintable(&message, body); err != nil {
			return smtpMessage{}, err
		}
	} else {
		boundary, err := generateMIMEBoundary()
		if err != nil {
			return smtpMessage{}, fmt.Errorf("generate MIME boundary: %w", err)
		}
		fmt.Fprintf(&message, "Content-Type: multipart/alternative; boundary=%q\r\n\r\n", boundary)

		// Least-preferred part first, as RFC 2046 §5.1.4 requires: clients render the
		// last part they understand, so HTML has to come after plain text.
		fmt.Fprintf(&message, "--%s\r\n", boundary)
		fmt.Fprint(&message, "Content-Type: text/plain; charset=UTF-8\r\n"+
			"Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		if err := writeQuotedPrintable(&message, plain); err != nil {
			return smtpMessage{}, err
		}
		fmt.Fprintf(&message, "\r\n--%s\r\n", boundary)
		fmt.Fprint(&message, "Content-Type: text/html; charset=UTF-8\r\n"+
			"Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		if err := writeQuotedPrintable(&message, body); err != nil {
			return smtpMessage{}, err
		}
		fmt.Fprintf(&message, "\r\n--%s--\r\n", boundary)
	}

	return smtpMessage{
		envelopeFrom: fromAddress.Address,
		envelopeTo:   recipientAddress.Address,
		data:         message.Bytes(),
	}, nil
}

func writeQuotedPrintable(dst *bytes.Buffer, content string) error {
	w := quotedprintable.NewWriter(dst)
	if _, err := w.Write([]byte(content)); err != nil {
		return fmt.Errorf("encode email body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close email body encoder: %w", err)
	}
	return nil
}

// generateMIMEBoundary returns a random boundary. Random rather than fixed so it
// cannot collide with body content, which would truncate the message.
func generateMIMEBoundary() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return "sub2api-" + hex.EncodeToString(raw), nil
}

func parseSMTPAddress(value, field string) (*mail.Address, error) {
	if strings.ContainsAny(value, "\r\n") {
		return nil, fmt.Errorf("invalid SMTP %s address: contains a line break", field)
	}

	cleaned := strings.TrimSpace(value)
	address, err := mail.ParseAddress(cleaned)
	if err != nil || strings.TrimSpace(address.Address) == "" {
		if err == nil {
			err = fmt.Errorf("address is empty")
		}
		return nil, fmt.Errorf("invalid SMTP %s address: %w", field, err)
	}
	return address, nil
}

func generateEmailMessageID(fromAddress, smtpHost string) (string, error) {
	randomID := make([]byte, 16)
	if _, err := rand.Read(randomID); err != nil {
		return "", err
	}

	domain := strings.TrimSpace(sanitizeEmailHeader(smtpHost))
	if at := strings.LastIndexByte(fromAddress, '@'); at >= 0 && at < len(fromAddress)-1 {
		domain = fromAddress[at+1:]
	}
	domain = strings.Trim(domain, "[]<>")
	if domain == "" {
		domain = "localhost"
	}

	return fmt.Sprintf("<%s@%s>", hex.EncodeToString(randomID), domain), nil
}
