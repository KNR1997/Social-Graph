// Package security holds the cryptographic adapters behind the application
// layer's PasswordHasher and TokenGenerator ports. It is the only package in
// the module that reaches for crypto/rand or a KDF, for the same reason that
// system is the only package that calls time.Now: one place to audit, one place
// to change the parameters.
package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"runtime"
	"strings"

	"golang.org/x/crypto/argon2"

	"github.com/kethaka-creskit/go-ddd-service/internal/application"
)

// Argon2id parameters. These are the RFC 9106 second-recommended settings:
// 64 MiB of memory, three passes. Memory is the cost that actually hurts an
// attacker with GPUs, which is why it is high relative to the iteration count.
//
// Raising any of these is safe at any time: the parameters are recorded in each
// stored digest, so old passwords keep verifying under the settings they were
// created with and are upgraded the next time their owner changes them.
const (
	argonMemoryKiB  = 64 * 1024
	argonIterations = 3
	argonSaltLen    = 16
	argonKeyLen     = 32

	// maxKeyLen bounds the digest length a stored hash may claim, so a corrupt
	// row cannot ask the KDF for an absurd allocation.
	maxKeyLen = 1024
)

// encodedSegments is the field count of the PHC string this package writes:
// "", "argon2id", "v=19", "m=…,t=…,p=…", salt, hash.
const encodedSegments = 6

// Hasher implements application.PasswordHasher with Argon2id.
type Hasher struct {
	parallelism uint8
}

var _ application.PasswordHasher = (*Hasher)(nil)

// NewHasher builds the password hasher.
//
// Parallelism follows the machine rather than being pinned, since it is the one
// parameter whose best value is a property of the host. It is recorded in the
// digest, so a digest written on an eight-core box still verifies on a
// single-core one.
func NewHasher() *Hasher {
	lanes := min(runtime.NumCPU(), int(^uint8(0)))

	return &Hasher{parallelism: uint8(lanes)} //nolint:gosec // clamped to the uint8 range on the line above.
}

// Hash derives a digest in the PHC string format, so the parameters and salt
// travel with the hash and nothing outside this file has to know them.
func (h *Hasher) Hash(plain string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("security - Hash - rand.Read: %w", err)
	}

	key := argon2.IDKey([]byte(plain), salt, argonIterations, argonMemoryKiB, h.parallelism, argonKeyLen)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argonMemoryKiB,
		argonIterations,
		h.parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// Verify recomputes the digest under the parameters recorded in hash and
// compares in constant time.
//
// A wrong password is application.ErrPasswordMismatch; anything else -- an
// unparseable digest, an algorithm this build does not know -- is a real error,
// so a corrupted row cannot quietly present as "wrong password" for everyone.
func (h *Hasher) Verify(hash, plain string) error {
	params, salt, want, err := decodeHash(hash)
	if err != nil {
		return err
	}

	// Bound the recorded key length before widening it: the digest is data read
	// back out of a database, and a nonsense length must be a parse error rather
	// than an allocation request.
	if len(want) == 0 || len(want) > maxKeyLen {
		return fmt.Errorf("%w: digest length %d", errMalformedHash, len(want))
	}

	keyLen := uint32(len(want)) //nolint:gosec // bounded to maxKeyLen on the line above.

	got := argon2.IDKey([]byte(plain), salt, params.iterations, params.memory, params.parallelism, keyLen)

	if subtle.ConstantTimeCompare(got, want) != 1 {
		return application.ErrPasswordMismatch
	}

	return nil
}

// argonParams are the cost settings read back out of a stored digest.
type argonParams struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
}

// errMalformedHash is returned for a stored digest this package cannot read.
var errMalformedHash = errors.New("security: malformed password hash")

// decodeHash parses the PHC string written by Hash.
func decodeHash(encoded string) (params argonParams, salt, key []byte, err error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != encodedSegments || parts[1] != "argon2id" {
		return argonParams{}, nil, nil, fmt.Errorf("%w: unrecognised format", errMalformedHash)
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return argonParams{}, nil, nil, fmt.Errorf("%w: bad version: %w", errMalformedHash, err)
	}

	if version != argon2.Version {
		return argonParams{}, nil, nil, fmt.Errorf("%w: unsupported version %d", errMalformedHash, version)
	}

	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d",
		&params.memory, &params.iterations, &params.parallelism); err != nil {
		return argonParams{}, nil, nil, fmt.Errorf("%w: bad parameters: %w", errMalformedHash, err)
	}

	salt, err = base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil {
		return argonParams{}, nil, nil, fmt.Errorf("%w: bad salt: %w", errMalformedHash, err)
	}

	key, err = base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil {
		return argonParams{}, nil, nil, fmt.Errorf("%w: bad digest: %w", errMalformedHash, err)
	}

	return params, salt, key, nil
}
