package domain

import (
	"fmt"
	"strings"
	"time"
)

// Customer is the owner of the vehicles the workshop services.
type Customer struct {
	ID             string
	FullName       string
	DocumentNumber string
	Phone          string
	Email          string
	CreatedAt      time.Time
}

// NewCustomer builds a customer after validating its contact data.
func NewCustomer(id, fullName, documentNumber, phone, email string, createdAt time.Time) (Customer, error) {
	fullName = strings.TrimSpace(fullName)
	documentNumber = strings.TrimSpace(documentNumber)
	phone = strings.TrimSpace(phone)
	email = strings.TrimSpace(strings.ToLower(email))
	if id == "" {
		return Customer{}, fmt.Errorf("%w: customer identifier is required", ErrInvalidInput)
	}
	if fullName == "" {
		return Customer{}, fmt.Errorf("%w: customer name is required", ErrInvalidInput)
	}
	if documentNumber == "" {
		return Customer{}, fmt.Errorf("%w: customer document number is required", ErrInvalidInput)
	}
	if phone == "" {
		return Customer{}, fmt.Errorf("%w: customer phone is required", ErrInvalidInput)
	}
	if !strings.Contains(email, "@") || strings.HasPrefix(email, "@") || strings.HasSuffix(email, "@") {
		return Customer{}, fmt.Errorf("%w: customer email is not a valid address", ErrInvalidInput)
	}
	if hasUnsafeMarkup(fullName) {
		return Customer{}, fmt.Errorf("%w: customer name carries unsafe characters", ErrInvalidInput)
	}
	if hasUnsafeMarkup(documentNumber) {
		return Customer{}, fmt.Errorf("%w: customer document number carries unsafe characters", ErrInvalidInput)
	}
	if hasUnsafeMarkup(phone) {
		return Customer{}, fmt.Errorf("%w: customer phone carries unsafe characters", ErrInvalidInput)
	}
	return Customer{
		ID:             id,
		FullName:       fullName,
		DocumentNumber: documentNumber,
		Phone:          phone,
		Email:          email,
		CreatedAt:      createdAt,
	}, nil
}
