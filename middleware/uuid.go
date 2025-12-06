package middleware

import (
	"github.com/google/uuid"
)

// generateUUID generates a new UUID
func generateUUID() string {
	return uuid.New().String()
}

