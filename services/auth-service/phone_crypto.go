package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
)

// derivePhoneKey turns an arbitrary env secret into a 32-byte AES-256 key.
func derivePhoneKey(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}

// encryptPhoneE164 encrypts E.164 phone at rest (AES-GCM). Ciphertext is nonce||tag||ciphertext.
func encryptPhoneE164(e164 string, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, []byte(e164), nil), nil
}

// decryptPhoneE164 decrypts a phone blob produced by encryptPhoneE164.
func decryptPhoneE164(blob []byte, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(blob) < ns {
		return "", fmt.Errorf("ciphertext too short")
	}
	plain, err := gcm.Open(nil, blob[:ns], blob[ns:], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
