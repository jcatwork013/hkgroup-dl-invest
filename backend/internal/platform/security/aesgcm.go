package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
)

// Cryptor encrypts/decrypts bytes at rest with AES-256-GCM. Used for confidential KYC images (CCCD).
type Cryptor struct {
	gcm cipher.AEAD
}

// NewCryptor takes a 64-hex-char (32-byte) key.
func NewCryptor(hexKey string) (*Cryptor, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, errors.New("KYC_ENC_KEY must be hex")
	}
	if len(key) != 32 {
		return nil, errors.New("KYC_ENC_KEY must be 32 bytes (64 hex chars)")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cryptor{gcm: gcm}, nil
}

// Encrypt returns nonce||ciphertext.
func (c *Cryptor) Encrypt(plain []byte) ([]byte, error) {
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return c.gcm.Seal(nonce, nonce, plain, nil), nil
}

// Decrypt expects nonce||ciphertext.
func (c *Cryptor) Decrypt(data []byte) ([]byte, error) {
	ns := c.gcm.NonceSize()
	if len(data) < ns {
		return nil, errors.New("ciphertext too short")
	}
	return c.gcm.Open(nil, data[:ns], data[ns:], nil)
}
