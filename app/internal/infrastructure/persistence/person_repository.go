package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kethaka-creskit/go-ddd-service/internal/domain/person/entity"
	"github.com/kethaka-creskit/go-ddd-service/internal/infrastructure/persistence/sqlcgen"
)

// PersonRepository implements entity.Repository.
type PersonRepository struct {
	pool *pgxpool.Pool
}

// Compile-time proof that this adapter satisfies the domain port. Cheap, and it
// fails the build rather than the wiring when the interface changes.
var _ entity.Repository = (*PersonRepository)(nil)

// NewPersonRepository builds the repository.
func NewPersonRepository(pool *pgxpool.Pool) *PersonRepository {
	return &PersonRepository{pool: pool}
}

// Add inserts a new person.
func (r *PersonRepository) Add(ctx context.Context, p *entity.Person) error {
	d := p.Details()

	err := queries(ctx, r.pool).InsertPerson(ctx, sqlcgen.InsertPersonParams{
		ID:           p.ID(),
		UserID:       p.UserID(),
		IsSelf:       p.IsSelf(),
		FullName:     d.FullName,
		Email:        optionalText(d.Email),
		Phone:        optionalText(d.Phone),
		Occupation:   optionalText(d.Occupation),
		Organization: optionalText(d.Organization),
		Location:     optionalText(d.Location),
		Notes:        optionalText(d.Notes),
		Version:      p.Version(),
		CreatedAt:    p.CreatedAt(),
		UpdatedAt:    p.UpdatedAt(),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		// The only unique constraints on this table are the primary key and
		// one-self-person-per-user, so a duplicate here is a racing writer
		// rather than bad input.
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return fmt.Errorf("persistence - Add: %w", entity.ErrConflict)
		}

		return fmt.Errorf("persistence - Add - InsertPerson: %w", err)
	}

	return nil
}

// Update writes changed details, guarded by the version the person was loaded
// at and by the owner.
func (r *PersonRepository) Update(ctx context.Context, p *entity.Person) error {
	d := p.Details()

	stored, err := queries(ctx, r.pool).UpdatePerson(ctx, sqlcgen.UpdatePersonParams{
		ID:           p.ID(),
		UserID:       p.UserID(),
		FullName:     d.FullName,
		Email:        optionalText(d.Email),
		Phone:        optionalText(d.Phone),
		Occupation:   optionalText(d.Occupation),
		Organization: optionalText(d.Organization),
		Location:     optionalText(d.Location),
		Notes:        optionalText(d.Notes),
		UpdatedAt:    p.UpdatedAt(),
		Version:      p.Version(),
	})
	if err != nil {
		// No row coming back means the id is gone, the caller does not own it,
		// or another writer moved the version on. All three are conflicts from
		// here; the caller already established that the row existed by loading
		// it.
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("persistence - Update: %w", entity.ErrConflict)
		}

		return fmt.Errorf("persistence - Update - UpdatePerson: %w", err)
	}

	// The aggregate now carries the version the database assigned, so the
	// response reports a version the client can write with next time.
	p.MarkStored(stored)

	return nil
}

// Delete removes a person. The row is deleted outright rather than flagged, and
// the schema decides the blast radius: relationships and their interactions
// cascade away, while anyone introduced through this person keeps their history
// with met_through_person_id set to null.
func (r *PersonRepository) Delete(ctx context.Context, userID, id uuid.UUID) error {
	rows, err := queries(ctx, r.pool).DeletePerson(ctx, sqlcgen.DeletePersonParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		return fmt.Errorf("persistence - Delete - DeletePerson: %w", err)
	}

	// The statement also refuses to delete the owner's self person, so a zero
	// here can mean "no such person", "not yours" or "that is your own node".
	// All three are reported as not found: the API never confirms that an id
	// exists in another account, and the self node is not a person the owner
	// can act on through this endpoint.
	if rows == 0 {
		return fmt.Errorf("persistence - Delete: %w", entity.ErrNotFound)
	}

	return nil
}

// ByID loads one person owned by userID.
func (r *PersonRepository) ByID(ctx context.Context, userID, id uuid.UUID) (*entity.Person, error) {
	row, err := queries(ctx, r.pool).GetPerson(ctx, sqlcgen.GetPersonParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("persistence - ByID: %w", entity.ErrNotFound)
		}

		return nil, fmt.Errorf("persistence - ByID - GetPerson: %w", err)
	}

	return toPersonEntity(&row), nil
}

// BySelf loads the owner's own node in the graph.
func (r *PersonRepository) BySelf(ctx context.Context, userID uuid.UUID) (*entity.Person, error) {
	row, err := queries(ctx, r.pool).GetSelfPerson(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("persistence - BySelf: %w", entity.ErrNotFound)
		}

		return nil, fmt.Errorf("persistence - BySelf - GetSelfPerson: %w", err)
	}

	return toPersonEntity(&row), nil
}

// List returns a page of the owner's people, newest first, optionally narrowed
// by a search term.
func (r *PersonRepository) List(
	ctx context.Context,
	userID uuid.UUID,
	filter entity.Filter,
) ([]*entity.Person, error) {
	rows, err := queries(ctx, r.pool).ListPersons(ctx, sqlcgen.ListPersonsParams{
		UserID: userID,
		Search: optionalText(filter.Search),
		Limit:  filter.Page.Limit,
		Offset: filter.Page.Offset,
	})
	if err != nil {
		return nil, fmt.Errorf("persistence - List - ListPersons: %w", err)
	}

	persons := make([]*entity.Person, 0, len(rows))
	// Index rather than range-copy: the generated row struct is large enough
	// that copying it per iteration shows up in profiles on wide pages.
	for i := range rows {
		persons = append(persons, toPersonEntity(&rows[i]))
	}

	return persons, nil
}

// Count returns how many people match the same filter List applies, so a
// paginated response can report a total.
func (r *PersonRepository) Count(
	ctx context.Context,
	userID uuid.UUID,
	filter entity.Filter,
) (int64, error) {
	total, err := queries(ctx, r.pool).CountPersons(ctx, sqlcgen.CountPersonsParams{
		UserID: userID,
		Search: optionalText(filter.Search),
	})
	if err != nil {
		return 0, fmt.Errorf("persistence - Count - CountPersons: %w", err)
	}

	return total, nil
}

// toPersonEntity maps a row to the aggregate. This is the anti-corruption seam:
// the column layout, and the nullability that forces pgtype here, stop at this
// function and never reach the domain.
//
// Unlike the account mapping this cannot fail. A person has no value object
// that can reject a stored value, and Reconstitute deliberately does not
// revalidate, so every stored row loads.
func toPersonEntity(row *sqlcgen.Person) *entity.Person {
	return entity.Reconstitute(
		row.ID,
		row.UserID,
		row.IsSelf,
		entity.Details{
			FullName:     row.FullName,
			Email:        textOrEmpty(row.Email),
			Phone:        textOrEmpty(row.Phone),
			Occupation:   textOrEmpty(row.Occupation),
			Organization: textOrEmpty(row.Organization),
			Location:     textOrEmpty(row.Location),
			Notes:        textOrEmpty(row.Notes),
		},
		row.Version,
		row.CreatedAt,
		row.UpdatedAt,
	)
}

// optionalText maps a domain field, where an empty string means absent, to a
// nullable column. Storing NULL rather than an empty string keeps one
// representation of "not recorded" in the database.
func optionalText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}

	return pgtype.Text{String: s, Valid: true}
}

// textOrEmpty is the inverse: a NULL column reads back as an empty string, so
// the domain never has to carry a pointer or a validity flag.
func textOrEmpty(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}

	return t.String
}
