package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
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

// GetDefaultIterations returns the default PBKDF2 iterations, configurable via env
func GetDefaultIterations() int {
	// This will be overridden by config, but provides a default
	return 600000
}

// Encrypt encrypts sensitive data using AES-GCM
func Encrypt(plaintext string, key string) (string, error) {
	return EncryptWithIterations(plaintext, key, GetDefaultIterations())
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
	return DecryptWithIterations(encrypted, key, GetDefaultIterations())
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

// HashCardNumber creates a salted hash of card number for verification
func HashCardNumber(cardNumber string) string {
	// Generate a random salt for each card number
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		// Fallback: use a deterministic salt if random generation fails
		salt = []byte("fallback-salt-16b")
	}

	// Use PBKDF2 with SHA-256 for key derivation (similar to encryption)
	hash := pbkdf2.Key([]byte(cardNumber), salt, 100000, 32, sha256.New)
	
	// Combine salt and hash for storage
	result := make([]byte, 16+32)
	copy(result[:16], salt)
	copy(result[16:], hash)
	
	return base64.StdEncoding.EncodeToString(result)
}

// VerifyCardNumberHash verifies a card number against a stored hash
func VerifyCardNumberHash(cardNumber, storedHash string) bool {
	data, err := base64.StdEncoding.DecodeString(storedHash)
	if err != nil || len(data) < 48 {
		return false
	}

	salt := data[:16]
	expectedHash := data[16:]

	hash := pbkdf2.Key([]byte(cardNumber), salt, 100000, 32, sha256.New)
	
	return subtle.ConstantTimeCompare(hash, expectedHash) == 1
}

