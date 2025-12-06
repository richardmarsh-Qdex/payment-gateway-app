package utils

import (
	"regexp"
	"strconv"
	"strings"
)

// ValidateCardNumber validates credit card number using Luhn algorithm
func ValidateCardNumber(cardNumber string) bool {
	cardNumber = strings.ReplaceAll(cardNumber, " ", "")
	cardNumber = strings.ReplaceAll(cardNumber, "-", "")

	if matched, _ := regexp.MatchString(`^\d+$`, cardNumber); !matched {
		return false
	}

	if len(cardNumber) < 13 || len(cardNumber) > 19 {
		return false
	}

	sum := 0
	isEven := false

	for i := len(cardNumber) - 1; i >= 0; i-- {
		digit, _ := strconv.Atoi(string(cardNumber[i]))

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
	matched, _ := regexp.MatchString(`^\d{3,4}$`, cvv)
	return matched
}

// ValidateExpiry validates expiry date (MM/YY or MM/YYYY format)
func ValidateExpiry(expiry string) bool {
	matched, _ := regexp.MatchString(`^(0[1-9]|1[0-2])\/(\d{2}|\d{4})$`, expiry)
	return matched
}

// ValidateEmail validates email address
func ValidateEmail(email string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`, email)
	return matched
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

