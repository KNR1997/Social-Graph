// Package entity holds the account aggregate. Every invariant about an account
// lives in this package and nowhere else.
//
// This package must not import anything from the rest of the module: no config,
// no logger, no database, no HTTP. google/uuid is the one external type allowed
// in, as a plain value type for identity.
package entity

import (
	"errors"
	"fmt"
	"strings"
)

// Money errors.
var (
	ErrInvalidCurrency  = errors.New("currency must be a 3-letter code")
	ErrCurrencyMismatch = errors.New("currency mismatch")
	ErrNotPositive      = errors.New("amount must be positive")
)

// Currency is an ISO-4217 alphabetic code, uppercase.
type Currency string

// ParseCurrency validates and normalises a currency code.
func ParseCurrency(s string) (Currency, error) {
	c := Currency(strings.ToUpper(strings.TrimSpace(s)))
	if len(c) != currencyCodeLen {
		return "", fmt.Errorf("%w: %q", ErrInvalidCurrency, s)
	}
	for _, r := range c {
		if r < 'A' || r > 'Z' {
			return "", fmt.Errorf("%w: %q", ErrInvalidCurrency, s)
		}
	}
	return c, nil
}

// Money is an immutable value object. The amount is held in the currency's
// minor unit (cents for USD) as an integer: never use floating point for money.
type Money struct {
	minor    int64
	currency Currency
}

// NewMoney builds a Money value, validating the currency.
func NewMoney(minor int64, currency string) (Money, error) {
	c, err := ParseCurrency(currency)
	if err != nil {
		return Money{}, err
	}
	return Money{minor: minor, currency: c}, nil
}

// Zero returns a zero amount in an already-validated currency.
func Zero(c Currency) Money {
	return Money{minor: 0, currency: c}
}

// Minor returns the amount in the currency minor unit.
func (m Money) Minor() int64 { return m.minor }

// Currency returns the currency code.
func (m Money) Currency() Currency { return m.currency }

// IsZero reports whether the amount is exactly zero.
func (m Money) IsZero() bool { return m.minor == 0 }

// IsPositive reports whether the amount is greater than zero.
func (m Money) IsPositive() bool { return m.minor > 0 }

// IsNegative reports whether the amount is less than zero.
func (m Money) IsNegative() bool { return m.minor < 0 }

// Add returns the sum. Currencies must match.
func (m Money) Add(o Money) (Money, error) {
	if err := m.sameCurrency(o); err != nil {
		return Money{}, err
	}
	return Money{minor: m.minor + o.minor, currency: m.currency}, nil
}

// Sub returns the difference. Currencies must match. The result may be
// negative: rejecting that is the aggregate's job, not the value object's.
func (m Money) Sub(o Money) (Money, error) {
	if err := m.sameCurrency(o); err != nil {
		return Money{}, err
	}
	return Money{minor: m.minor - o.minor, currency: m.currency}, nil
}

func (m Money) sameCurrency(o Money) error {
	if m.currency != o.currency {
		return fmt.Errorf("%w: %s vs %s", ErrCurrencyMismatch, m.currency, o.currency)
	}
	return nil
}

func (m Money) String() string {
	return fmt.Sprintf("%d %s", m.minor, m.currency)
}
