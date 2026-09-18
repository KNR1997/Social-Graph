package entity

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Account errors. Outer layers match on these with errors.Is rather than
// inspecting strings, which is what lets the REST layer map them to statuses
// without knowing anything about the domain internals.
var (
	ErrEmptyOwner        = errors.New("owner must not be empty")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrNotActive         = errors.New("account is not active")
	ErrBalanceNotZero    = errors.New("account balance must be zero to close")
	ErrInvalidStatus     = errors.New("invalid account status")
	ErrNotFound          = errors.New("account not found")
	ErrConflict          = errors.New("account was modified concurrently")
)

// Status is the account lifecycle state.
type Status string

// The account lifecycle states.
const (
	// StatusActive accounts accept deposits and withdrawals.
	StatusActive Status = "active"
	// StatusFrozen accounts reject balance changes but can be reactivated.
	StatusFrozen Status = "frozen"
	// StatusClosed accounts are terminal.
	StatusClosed Status = "closed"
)

// ParseStatus validates a persisted status string.
func ParseStatus(s string) (Status, error) {
	switch Status(s) {
	case StatusActive, StatusFrozen, StatusClosed:
		return Status(s), nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidStatus, s)
	}
}

const (
	maxOwnerLen = 200

	// currencyCodeLen is the length of an ISO-4217 alphabetic code.
	currencyCodeLen = 3
)

// Account is the aggregate root. All fields are unexported: the only way to
// change one is through a method that enforces the invariants, so an Account
// value can never be observed in an invalid state.
//
// Invariants:
//   - owner is non-empty and at most maxOwnerLen characters
//   - balance is never negative
//   - balance currency never changes after opening
//   - only an active account can be debited or credited
//   - an account can only be closed with a zero balance
type Account struct {
	id        uuid.UUID
	owner     string
	balance   Money
	status    Status
	version   int64
	createdAt time.Time
	updatedAt time.Time
}

// Open creates a new account. The id and the clock are passed in rather than
// generated here, which keeps this package free of dependencies and makes every
// test deterministic.
func Open(id uuid.UUID, owner, currency string, now time.Time) (*Account, error) {
	owner = strings.TrimSpace(owner)
	if owner == "" {
		return nil, ErrEmptyOwner
	}
	if len(owner) > maxOwnerLen {
		return nil, fmt.Errorf("%w: longer than %d characters", ErrEmptyOwner, maxOwnerLen)
	}

	c, err := ParseCurrency(currency)
	if err != nil {
		return nil, err
	}

	return &Account{
		id:        id,
		owner:     owner,
		balance:   Zero(c),
		status:    StatusActive,
		version:   1,
		createdAt: now.UTC(),
		updatedAt: now.UTC(),
	}, nil
}

// Reconstitute rebuilds an Account from persisted state. It deliberately skips
// the Open invariants: the row was valid when it was written, and re-validating
// would make old rows unloadable after a rule change. Only the repository
// should call this.
func Reconstitute(
	id uuid.UUID,
	owner string,
	balance Money,
	status Status,
	version int64,
	createdAt, updatedAt time.Time,
) *Account {
	return &Account{
		id:        id,
		owner:     owner,
		balance:   balance,
		status:    status,
		version:   version,
		createdAt: createdAt.UTC(),
		updatedAt: updatedAt.UTC(),
	}
}

// ID returns the account identifier.
func (a *Account) ID() uuid.UUID { return a.id }

// Owner returns the account owner.
func (a *Account) Owner() string { return a.owner }

// Balance returns the current balance.
func (a *Account) Balance() Money { return a.balance }

// Status returns the lifecycle state.
func (a *Account) Status() Status { return a.status }

// Version returns the version the account was loaded at, used for optimistic
// locking by the repository.
func (a *Account) Version() int64 { return a.version }

// CreatedAt returns when the account was opened.
func (a *Account) CreatedAt() time.Time { return a.createdAt }

// UpdatedAt returns when the account last changed.
func (a *Account) UpdatedAt() time.Time { return a.updatedAt }

// Deposit credits the account.
func (a *Account) Deposit(m Money, now time.Time) error {
	if err := a.requireActive(); err != nil {
		return err
	}
	if !m.IsPositive() {
		return fmt.Errorf("%w: deposit of %s", ErrNotPositive, m)
	}

	balance, err := a.balance.Add(m)
	if err != nil {
		return err
	}

	a.balance = balance
	a.updatedAt = now.UTC()

	return nil
}

// Withdraw debits the account, refusing to overdraw it.
func (a *Account) Withdraw(m Money, now time.Time) error {
	if err := a.requireActive(); err != nil {
		return err
	}
	if !m.IsPositive() {
		return fmt.Errorf("%w: withdrawal of %s", ErrNotPositive, m)
	}

	balance, err := a.balance.Sub(m)
	if err != nil {
		return err
	}
	if balance.IsNegative() {
		return fmt.Errorf("%w: balance %s, requested %s", ErrInsufficientFunds, a.balance, m)
	}

	a.balance = balance
	a.updatedAt = now.UTC()

	return nil
}

// Freeze suspends transacting on the account. Freezing a frozen account is a
// no-op so that retried commands stay idempotent.
func (a *Account) Freeze(now time.Time) error {
	switch a.status {
	case StatusFrozen:
		return nil
	case StatusClosed:
		return fmt.Errorf("%w: %s", ErrNotActive, a.status)
	case StatusActive:
		// The only state that can be frozen; handled below.
	}

	a.status = StatusFrozen
	a.updatedAt = now.UTC()

	return nil
}

// Unfreeze returns a frozen account to active.
func (a *Account) Unfreeze(now time.Time) error {
	switch a.status {
	case StatusActive:
		return nil
	case StatusClosed:
		return fmt.Errorf("%w: %s", ErrNotActive, a.status)
	case StatusFrozen:
		// The only state that can be unfrozen; handled below.
	}

	a.status = StatusActive
	a.updatedAt = now.UTC()

	return nil
}

// Close permanently closes the account. Funds must be withdrawn first.
func (a *Account) Close(now time.Time) error {
	if a.status == StatusClosed {
		return nil
	}
	if !a.balance.IsZero() {
		return fmt.Errorf("%w: balance is %s", ErrBalanceNotZero, a.balance)
	}

	a.status = StatusClosed
	a.updatedAt = now.UTC()

	return nil
}

func (a *Account) requireActive() error {
	if a.status != StatusActive {
		return fmt.Errorf("%w: %s", ErrNotActive, a.status)
	}
	return nil
}
