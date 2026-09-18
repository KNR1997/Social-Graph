// Package entity holds the identity aggregates: the user who signs in and the
// session that proves they did. They share a package because they share one
// bounded context -- a session is meaningless without the user it belongs to,
// and every rule about "is this caller authenticated" needs both.
//
// This package must not import anything from the rest of the module: no config,
// no logger, no database, no HTTP. It also holds no cryptography. Hashing a
// password and minting a token are technical concerns satisfied by ports the
// application layer declares; what lives here is the policy about them (how
// long a password must be, when a session has expired).
package entity

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Identity errors. Outer layers match on these with errors.Is rather than
// inspecting strings, which is what lets the REST layer map them to statuses
// without knowing anything about the domain internals.
var (
	ErrInvalidEmail       = errors.New("email is not a valid address")
	ErrEmptyName          = errors.New("name must not be empty")
	ErrWeakPassword       = errors.New("password does not meet the policy")
	ErrInvalidRole        = errors.New("invalid role")
	ErrEmailTaken         = errors.New("email is already registered")
	ErrInvalidCredentials = errors.New("email or password is incorrect")
	ErrUserNotFound       = errors.New("user not found")
	ErrConflict           = errors.New("user was modified concurrently")
)

// Password policy. These are deliberately domain constants rather than
// configuration: a deployment that can weaken them is a deployment that will.
const (
	// MinPasswordLen is the shortest password the service accepts.
	MinPasswordLen = 8
	// MaxPasswordLen caps the input before it reaches the hasher. Argon2 will
	// happily consume a megabyte of "password" and charge the server for it.
	MaxPasswordLen = 128
)

const (
	maxNameLen  = 200
	maxEmailLen = 254 // RFC 5321 limit on a forward path.
)

// Email is a normalised address. Two users cannot share one, so it is compared
// and stored lowercase: Ada@example.com and ada@example.com are one identity.
type Email string

// ParseEmail validates and normalises an address.
func ParseEmail(s string) (Email, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" || len(s) > maxEmailLen {
		return "", fmt.Errorf("%w: %q", ErrInvalidEmail, s)
	}

	// ParseAddress accepts the display-name form ("Ada <ada@example.com>"),
	// which is not what anyone means by an account identifier.
	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Address != s {
		return "", fmt.Errorf("%w: %q", ErrInvalidEmail, s)
	}

	return Email(s), nil
}

// String returns the normalised address.
func (e Email) String() string { return string(e) }

// Role is what a user is allowed to do. It is coarse on purpose: fine-grained
// permissions belong to whatever the feature actually needs, not to a
// speculative matrix invented up front.
type Role string

// The roles.
const (
	// RoleMember is the default: full access to the caller's own data.
	RoleMember Role = "member"
	// RoleAdmin additionally reaches administrative views.
	RoleAdmin Role = "admin"
)

// ParseRole validates a persisted or supplied role string.
func ParseRole(s string) (Role, error) {
	switch Role(s) {
	case RoleMember, RoleAdmin:
		return Role(s), nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidRole, s)
	}
}

// ValidatePassword applies the password policy to a plaintext password.
//
// It lives in the domain and takes plaintext because the policy is a business
// rule, not a cryptographic one: the application layer calls this before
// handing the password to the PasswordHasher port, and this package never sees
// the resulting digest's format.
func ValidatePassword(plain string) error {
	// Count runes, not bytes: a passphrase of eight non-ASCII characters is
	// eight characters, whatever it weighs.
	n := utf8.RuneCountInString(plain)

	switch {
	case n < MinPasswordLen:
		return fmt.Errorf("%w: must be at least %d characters", ErrWeakPassword, MinPasswordLen)
	case n > MaxPasswordLen:
		return fmt.Errorf("%w: must be at most %d characters", ErrWeakPassword, MaxPasswordLen)
	default:
		return nil
	}
}

// User is the identity aggregate root. All fields are unexported: the only way
// to change one is through a method that enforces the invariants, so a User
// value can never be observed in an invalid state.
//
// Invariants:
//   - email is a normalised, valid address, and is the identity
//   - name is non-empty and at most maxNameLen characters
//   - passwordHash is never empty, and never holds a plaintext password
//   - role is one of the known roles
type User struct {
	id           uuid.UUID
	email        Email
	name         string
	passwordHash string
	role         Role
	version      int64
	createdAt    time.Time
	updatedAt    time.Time
}

// Register creates a new user.
//
// It takes an already-hashed password rather than a plaintext one. That is what
// keeps this package free of a cryptography dependency, and it means no code
// path can construct a User whose stored secret is readable: there is nowhere
// to put a plaintext string.
func Register(id uuid.UUID, email, name, passwordHash string, role Role, now time.Time) (*User, error) {
	addr, err := ParseEmail(email)
	if err != nil {
		return nil, err
	}

	name, err = validateName(name)
	if err != nil {
		return nil, err
	}

	if passwordHash == "" {
		return nil, fmt.Errorf("%w: empty hash", ErrWeakPassword)
	}

	if _, err := ParseRole(string(role)); err != nil {
		return nil, err
	}

	return &User{
		id:           id,
		email:        addr,
		name:         name,
		passwordHash: passwordHash,
		role:         role,
		version:      1,
		createdAt:    now.UTC(),
		updatedAt:    now.UTC(),
	}, nil
}

// ReconstituteUser rebuilds a user from storage without re-running the
// creation rules. The repository is trusted to return what it was given; the
// CHECK constraints in the migration are the backstop for anything that
// reached the table another way.
func ReconstituteUser(
	id uuid.UUID,
	email Email,
	name, passwordHash string,
	role Role,
	version int64,
	createdAt, updatedAt time.Time,
) *User {
	return &User{
		id:           id,
		email:        email,
		name:         name,
		passwordHash: passwordHash,
		role:         role,
		version:      version,
		createdAt:    createdAt.UTC(),
		updatedAt:    updatedAt.UTC(),
	}
}

// Rename changes the display name.
func (u *User) Rename(name string, now time.Time) error {
	name, err := validateName(name)
	if err != nil {
		return err
	}

	u.name = name
	u.updatedAt = now.UTC()

	return nil
}

// ChangePassword replaces the stored digest. Like Register it takes a hash, so
// the aggregate never handles a plaintext secret.
//
// Every existing session for this user should be discarded by the caller: a
// password change is how someone locks out whoever stole the old one, and it
// only does that if the thief's cookie stops working.
func (u *User) ChangePassword(passwordHash string, now time.Time) error {
	if passwordHash == "" {
		return fmt.Errorf("%w: empty hash", ErrWeakPassword)
	}

	u.passwordHash = passwordHash
	u.updatedAt = now.UTC()

	return nil
}

// ID returns the identifier.
func (u *User) ID() uuid.UUID { return u.id }

// Email returns the normalised address.
func (u *User) Email() Email { return u.email }

// Name returns the display name.
func (u *User) Name() string { return u.name }

// PasswordHash returns the stored digest. Only the persistence adapter and the
// PasswordHasher port have any business calling this; it is never serialised
// onto the wire.
func (u *User) PasswordHash() string { return u.passwordHash }

// Role returns the role.
func (u *User) Role() Role { return u.role }

// Version returns the optimistic-lock version.
func (u *User) Version() int64 { return u.version }

// CreatedAt returns the creation timestamp.
func (u *User) CreatedAt() time.Time { return u.createdAt }

// UpdatedAt returns the last-modified timestamp.
func (u *User) UpdatedAt() time.Time { return u.updatedAt }

// validateName trims and bounds a display name.
func validateName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrEmptyName
	}

	if utf8.RuneCountInString(name) > maxNameLen {
		return "", fmt.Errorf("%w: longer than %d characters", ErrEmptyName, maxNameLen)
	}

	return name, nil
}
