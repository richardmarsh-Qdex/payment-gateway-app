package utils

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	cvvRegex      = regexp.MustCompile(`^\d{3,4}$`)
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	digitsRegex   = regexp.MustCompile(`^\d+$`)
	expiryRegex   = regexp.MustCompile(`^(0[1-9]|1[0-2])\/(\d{2}|\d{4})$`)
)

// ValidateCardNumber validates credit card number using Luhn algorithm
func ValidateCardNumber(cardNumber string) bool {
	cardNumber = strings.ReplaceAll(cardNumber, " ", "")
	cardNumber = strings.ReplaceAll(cardNumber, "-", "")

	if !digitsRegex.MatchString(cardNumber) {
		return false
	}

	if len(cardNumber) < 13 || len(cardNumber) > 19 {
		return false
	}

	sum := 0
	isEven := false

	for i := len(cardNumber) - 1; i >= 0; i-- {
		digit, err := strconv.Atoi(string(cardNumber[i]))
		if err != nil {
			return false
		}

		if isEven {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		isEven = !isEven
	}

	return sum%10 == 0
}

// ValidateCVV validates CVV (3 or 4 digits)
func ValidateCVV(cvv string) bool {
	return cvvRegex.MatchString(cvv)
}

// ValidateExpiry validates expiry date (MM/YY or MM/YYYY format) and checks if it's expired
func ValidateExpiry(expiry string) bool {
	parts := strings.Split(expiry, "/")
	if len(parts) != 2 {
		return false
	}

	month, err := strconv.Atoi(parts[0])
	if err != nil || month < 1 || month > 12 {
		return false
	}

	year, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}

	if year < 100 {
		year += 2000
	}

	now := time.Now()
	lastDayOfExpiryMonth := time.Date(year, time.Month(month+1), 0, 23, 59, 59, 0, time.UTC)

	return !lastDayOfExpiryMonth.Before(now)
}

// ValidateEmail validates email address
func ValidateEmail(email string) bool {
	return emailRegex.MatchString(email)
}

// MaskCardNumber masks card number showing only last 4 digits
func MaskCardNumber(cardNumber string) string {
	cardNumber = strings.ReplaceAll(cardNumber, " ", "")
	cardNumber = strings.ReplaceAll(cardNumber, "-", "")
	
	if len(cardNumber) <= 4 {
		return strings.Repeat("*", len(cardNumber))
	}
	
	return strings.Repeat("*", len(cardNumber)-4) + cardNumber[len(cardNumber)-4:]
}

