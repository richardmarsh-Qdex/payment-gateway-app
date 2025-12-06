package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	saltSize  = 16
	nonceSize = 12
	keySize   = 32
)

var defaultIterations = 600000

// Encrypt encrypts sensitive data using AES-GCM
func Encrypt(plaintext string, key string) (string, error) {
	return EncryptWithIterations(plaintext, key, defaultIterations)
}

// EncryptWithIterations encrypts sensitive data using AES-GCM with custom iterations
func EncryptWithIterations(plaintext string, key string, iterations int) (string, error) {
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	derivedKey := pbkdf2.Key([]byte(key), salt, iterations, keySize, sha256.New)

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := aesGCM.Seal(nil, nonce, []byte(plaintext), nil)

	result := make([]byte, saltSize+nonceSize+len(ciphertext))
	copy(result[:saltSize], salt)
	copy(result[saltSize:saltSize+nonceSize], nonce)
	copy(result[saltSize+nonceSize:], ciphertext)

	return base64.StdEncoding.EncodeToString(result), nil
}

// Decrypt decrypts encrypted data
func Decrypt(encrypted string, key string) (string, error) {
	return DecryptWithIterations(encrypted, key, defaultIterations)
}

// DecryptWithIterations decrypts encrypted data with custom iterations
func DecryptWithIterations(encrypted string, key string, iterations int) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	if len(data) < saltSize+nonceSize {
		return "", fmt.Errorf("encrypted data too short")
	}

	salt := data[:saltSize]
	nonce := data[saltSize : saltSize+nonceSize]
	ciphertext := data[saltSize+nonceSize:]

	derivedKey := pbkdf2.Key([]byte(key), salt, iterations, keySize, sha256.New)

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// HashCardNumber creates a hash of card number for verification
func HashCardNumber(cardNumber string) string {
	hash := sha256.Sum256([]byte(cardNumber))
	return fmt.Sprintf("%x", hash)
}

