package domain_test

import (
	"errors"
	"testing"
	"time"

	"workshop/internal/domain"
)

func TestNewCustomerAcceptsValidContactData(t *testing.T) {
	customer, err := domain.NewCustomer("customer-1", "Ana Gomez", "CC-1000", "3001234567", "ana@example.com", time.Now())
	if err != nil {
		t.Fatalf("valid customer data must be accepted: %v", err)
	}
	if customer.FullName != "Ana Gomez" {
		t.Fatalf("the full name must be kept as given, got %q", customer.FullName)
	}
}

func TestNewCustomerRejectsAnHTMLPayloadInTheName(t *testing.T) {
	_, err := domain.NewCustomer(
		"customer-1", "O'Connor <script>alert(1)</script>", "CC-1000", "3001234567", "ana@example.com", time.Now(),
	)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("a name carrying HTML markup must be rejected as invalid input, got %v", err)
	}
}

func TestNewCustomerRejectsAnHTMLPayloadInTheDocumentNumber(t *testing.T) {
	_, err := domain.NewCustomer(
		"customer-1", "Ana Gomez", "DOC-999<script>", "3001234567", "ana@example.com", time.Now(),
	)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("a document number carrying an angle bracket must be rejected, got %v", err)
	}
}

// TestNewCustomerAcceptsASingleQuoteInTheName documents a deliberate choice: a
// SQL-style payload (quotes, "OR '1'='1") is not rejected here, because every
// repository query is parameterized and never concatenates this value into
// SQL. Blocking the quote character would also reject real names like
// "O'Connor". Rejecting only markup characters (<, >) covers the actual risk
// -- a stored value rendered unescaped outside the React frontend -- without
// refusing legitimate contact data.
func TestNewCustomerAcceptsASingleQuoteInTheName(t *testing.T) {
	_, err := domain.NewCustomer(
		"customer-1", "O'Connor", "DOC-999' OR '1'='1", "3001234567", "ana@example.com", time.Now(),
	)
	if err != nil {
		t.Fatalf("a single quote is not unsafe markup and must be accepted: %v", err)
	}
}
