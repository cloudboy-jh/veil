package app

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"filippo.io/age"
	"filippo.io/age/armor"
)

func encryptJSON(plaintext []byte, recipients []string) (string, error) {
	if len(recipients) == 0 {
		return "", fmt.Errorf("no age recipients configured")
	}
	ageRecipients := make([]age.Recipient, 0, len(recipients))
	for _, raw := range recipients {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		rec, err := age.ParseX25519Recipient(raw)
		if err != nil {
			continue
		}
		ageRecipients = append(ageRecipients, rec)
	}
	if len(ageRecipients) == 0 {
		return "", fmt.Errorf("no valid age recipients configured")
	}
	var out bytes.Buffer
	armored := armor.NewWriter(&out)
	w, err := age.Encrypt(armored, ageRecipients...)
	if err != nil {
		return "", fmt.Errorf("encrypt: %w", err)
	}
	if _, err := w.Write(plaintext); err != nil {
		return "", fmt.Errorf("encrypt write: %w", err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("encrypt close: %w", err)
	}
	if err := armored.Close(); err != nil {
		return "", fmt.Errorf("armor close: %w", err)
	}
	return out.String(), nil
}

func decryptJSON(ciphertext string, identity *age.X25519Identity) ([]byte, error) {
	r := armor.NewReader(strings.NewReader(ciphertext))
	dec, err := age.Decrypt(r, identity)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	b, err := io.ReadAll(dec)
	if err != nil {
		return nil, fmt.Errorf("decrypt read: %w", err)
	}
	return b, nil
}
