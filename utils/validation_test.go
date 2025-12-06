package utils

import "testing"

func TestValidateCardNumber(t *testing.T) {
	tests := []struct {
		name       string
		cardNumber string
		want       bool
	}{
		{
			name:       "Valid Visa",
			cardNumber: "4111111111111111",
			want:       true,
		},
		{
			name:       "Valid Mastercard",
			cardNumber: "5555555555554444",
			want:       true,
		},
		{
			name:       "Invalid card number",
			cardNumber: "1234567890123456",
			want:       false,
		},
		{
			name:       "Card with spaces",
			cardNumber: "4111 1111 1111 1111",
			want:       true,
		},
		{
			name:       "Card with dashes",
			cardNumber: "4111-1111-1111-1111",
			want:       true,
		},
		{
			name:       "Too short",
			cardNumber: "1234567890",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateCardNumber(tt.cardNumber); got != tt.want {
				t.Errorf("ValidateCardNumber() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateCVV(t *testing.T) {
	tests := []struct {
		name string
		cvv  string
		want bool
	}{
		{
			name: "Valid 3-digit CVV",
			cvv:  "123",
			want: true,
		},
		{
			name: "Valid 4-digit CVV",
			cvv:  "1234",
			want: true,
		},
		{
			name: "Invalid 2-digit CVV",
			cvv:  "12",
			want: false,
		},
		{
			name: "Invalid non-numeric",
			cvv:  "abc",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateCVV(tt.cvv); got != tt.want {
				t.Errorf("ValidateCVV() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{
			name:  "Valid email",
			email: "test@example.com",
			want:  true,
		},
		{
			name:  "Invalid email",
			email: "invalid-email",
			want:  false,
		},
		{
			name:  "Empty email",
			email: "",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateEmail(tt.email); got != tt.want {
				t.Errorf("ValidateEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMaskCardNumber(t *testing.T) {
	tests := []struct {
		name       string
		cardNumber string
		want       string
	}{
		{
			name:       "Standard card",
			cardNumber: "4111111111111111",
			want:       "************1111",
		},
		{
			name:       "Card with spaces",
			cardNumber: "4111 1111 1111 1111",
			want:       "************1111",
		},
		{
			name:       "Short card",
			cardNumber: "1234",
			want:       "1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaskCardNumber(tt.cardNumber); got != tt.want {
				t.Errorf("MaskCardNumber() = %v, want %v", got, tt.want)
			}
		})
	}
}

