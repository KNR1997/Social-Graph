// Package system holds the real implementations of the ambient ports that the
// application layer declares: the wall clock and the id source. They live here
// so that time.Now and uuid.New appear in exactly one place in the codebase.
package system

import (
	"time"

	"github.com/google/uuid"
)

// Clock implements application.Clock against the wall clock.
type Clock struct{}

// NewClock builds the system clock.
func NewClock() Clock { return Clock{} }

// Now returns the current UTC time.
func (Clock) Now() time.Time { return time.Now().UTC() }

// IDGenerator implements application.IDGenerator with random UUIDv4s.
type IDGenerator struct{}

// NewIDGenerator builds the id generator.
func NewIDGenerator() IDGenerator { return IDGenerator{} }

// NewID returns a fresh identifier.
func (IDGenerator) NewID() uuid.UUID { return uuid.New() }
