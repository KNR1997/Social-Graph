package entity

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound         = errors.New("person not found")
	ErrConflict         = errors.New("person was modified concurrently")
	ErrOwnerRequired    = errors.New("person owner is required")
	ErrFullNameRequired = errors.New("person full name is required")
	ErrInvalidEmail     = errors.New("person email is not a valid address")
	ErrFieldTooLong     = errors.New("person field is too long")
)

type FieldTooLongError struct {
	Field string
	Max   int
}

func (e *FieldTooLongError) Error() string {
	return fmt.Sprintf("%s must be at most %d characters", e.Field, e.Max)
}

func (e *FieldTooLongError) Is(target error) bool { return target == ErrFieldTooLong }

// Field bounds. These mirror the CHECK constraints in the persons migration.
// The domain is the source of truth and the constraints are defence in depth,
// so the two are changed together.
const (
	maxFullNameLen     = 200
	maxEmailLen        = 254
	maxPhoneLen        = 50
	maxOccupationLen   = 200
	maxOrganizationLen = 200
	maxLocationLen     = 200
	maxNotesLen        = 10000
)

type Details struct {
	FullName     string
	Email        string
	Phone        string
	Occupation   string
	Organization string
	Location     string
	Notes        string
}

type Person struct {
	id        uuid.UUID
	userID    uuid.UUID
	isSelf    bool
	details   Details
	version   int64
	createdAt time.Time
	updatedAt time.Time
}

func Create(
	id uuid.UUID,
	userID uuid.UUID,
	isSelf bool,
	details Details,
	now time.Time,
) (*Person, error) {
	if userID == uuid.Nil {
		return nil, ErrOwnerRequired
	}

	normalised, err := normaliseDetails(details)
	if err != nil {
		return nil, err
	}

	return &Person{
		id:        id,
		userID:    userID,
		isSelf:    isSelf,
		details:   normalised,
		version:   1,
		createdAt: now.UTC(),
		updatedAt: now.UTC(),
	}, nil
}

func Reconstitute(
	id uuid.UUID,
	userID uuid.UUID,
	isSelf bool,
	details Details,
	version int64,
	createdAt time.Time,
	updatedAt time.Time,
) *Person {
	return &Person{
		id:        id,
		userID:    userID,
		isSelf:    isSelf,
		details:   details,
		version:   version,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

func (p *Person) UpdateDetails(details Details, now time.Time) error {
	normalised, err := normaliseDetails(details)
	if err != nil {
		return err
	}

	p.details = normalised
	p.updatedAt = now.UTC()

	return nil
}

type DetailsPatch struct {
	FullName     *string
	Email        *string
	Phone        *string
	Occupation   *string
	Organization *string
	Location     *string
	Notes        *string
}

func (p *Person) ApplyPatch(patch DetailsPatch, now time.Time) error {
	merged := p.details

	for _, f := range []struct {
		from *string
		to   *string
	}{
		{patch.FullName, &merged.FullName},
		{patch.Email, &merged.Email},
		{patch.Phone, &merged.Phone},
		{patch.Occupation, &merged.Occupation},
		{patch.Organization, &merged.Organization},
		{patch.Location, &merged.Location},
		{patch.Notes, &merged.Notes},
	} {
		if f.from != nil {
			*f.to = *f.from
		}
	}

	return p.UpdateDetails(merged, now)
}

func normaliseDetails(d Details) (Details, error) {
	out := Details{
		FullName:     strings.TrimSpace(d.FullName),
		Email:        strings.ToLower(strings.TrimSpace(d.Email)),
		Phone:        strings.TrimSpace(d.Phone),
		Occupation:   strings.TrimSpace(d.Occupation),
		Organization: strings.TrimSpace(d.Organization),
		Location:     strings.TrimSpace(d.Location),
		Notes:        strings.TrimSpace(d.Notes),
	}

	if out.FullName == "" {
		return Details{}, ErrFullNameRequired
	}

	if err := checkEmail(out.Email); err != nil {
		return Details{}, err
	}

	bounds := []struct {
		name  string
		value string
		max   int
	}{
		{"full name", out.FullName, maxFullNameLen},
		{"phone", out.Phone, maxPhoneLen},
		{"occupation", out.Occupation, maxOccupationLen},
		{"organization", out.Organization, maxOrganizationLen},
		{"location", out.Location, maxLocationLen},
		{"notes", out.Notes, maxNotesLen},
	}
	for _, b := range bounds {
		if len(b.value) > b.max {
			return Details{}, &FieldTooLongError{Field: b.name, Max: b.max}
		}
	}

	return out, nil
}

func checkEmail(email string) error {
	if email == "" {
		return nil
	}

	if len(email) > maxEmailLen {
		return &FieldTooLongError{Field: "email", Max: maxEmailLen}
	}

	at := strings.IndexByte(email, '@')
	if at <= 0 || at != strings.LastIndexByte(email, '@') || at == len(email)-1 {
		return ErrInvalidEmail
	}

	if strings.ContainsAny(email, " \t\r\n") {
		return ErrInvalidEmail
	}

	return nil
}

func (p *Person) MarkStored(version int64) { p.version = version }

func (p *Person) ID() uuid.UUID { return p.id }

func (p *Person) UserID() uuid.UUID { return p.userID }

func (p *Person) IsSelf() bool { return p.isSelf }

func (p *Person) Details() Details { return p.details }

func (p *Person) FullName() string { return p.details.FullName }

func (p *Person) Email() string { return p.details.Email }

func (p *Person) Phone() string { return p.details.Phone }

func (p *Person) Occupation() string { return p.details.Occupation }

func (p *Person) Organization() string { return p.details.Organization }

func (p *Person) Location() string { return p.details.Location }

func (p *Person) Notes() string { return p.details.Notes }

func (p *Person) Version() int64 { return p.version }

func (p *Person) CreatedAt() time.Time { return p.createdAt }

func (p *Person) UpdatedAt() time.Time { return p.updatedAt }
