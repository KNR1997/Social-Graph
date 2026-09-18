package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kethaka-creskit/go-ddd-service/internal/domain/auth/entity"
	"github.com/kethaka-creskit/go-ddd-service/internal/infrastructure/persistence/sqlcgen"
)

// UserRepository is the Postgres adapter for entity.UserRepository.
type UserRepository struct {
	pool *pgxpool.Pool
}

var _ entity.UserRepository = (*UserRepository)(nil)

// NewUserRepository builds the repository.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Add inserts a newly registered user.
//
// A unique violation is the registration race: two requests both found the
// address free, and the index arbitrated. It maps to ErrEmailTaken so the
// loser sees the same message it would have seen a millisecond earlier.
func (r *UserRepository) Add(ctx context.Context, u *entity.User) error {
	err := queries(ctx, r.pool).InsertUser(ctx, sqlcgen.InsertUserParams{
		ID:           u.ID(),
		Email:        u.Email().String(),
		Name:         u.Name(),
		PasswordHash: u.PasswordHash(),
		Role:         string(u.Role()),
		Version:      u.Version(),
		CreatedAt:    u.CreatedAt(),
		UpdatedAt:    u.UpdatedAt(),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return fmt.Errorf("persistence - Add: %w", entity.ErrEmailTaken)
		}

		return fmt.Errorf("persistence - Add - InsertUser: %w", err)
	}

	return nil
}

// Update writes a user back under the optimistic lock.
func (r *UserRepository) Update(ctx context.Context, u *entity.User) error {
	rows, err := queries(ctx, r.pool).UpdateUser(ctx, sqlcgen.UpdateUserParams{
		ID:           u.ID(),
		Name:         u.Name(),
		PasswordHash: u.PasswordHash(),
		UpdatedAt:    u.UpdatedAt(),
		Version:      u.Version(),
	})
	if err != nil {
		return fmt.Errorf("persistence - Update - UpdateUser: %w", err)
	}

	// Zero rows means either the id is gone or another writer moved the version
	// on. Both are conflicts from this caller's point of view.
	if rows == 0 {
		return fmt.Errorf("persistence - Update: %w", entity.ErrConflict)
	}

	return nil
}

// ByID loads a user by identifier.
func (r *UserRepository) ByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	row, err := queries(ctx, r.pool).GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("persistence - ByID: %w", entity.ErrUserNotFound)
		}

		return nil, fmt.Errorf("persistence - ByID - GetUser: %w", err)
	}

	user, err := toUserEntity(&row)
	if err != nil {
		return nil, fmt.Errorf("persistence - ByID: %w", err)
	}

	return user, nil
}

// ByEmail loads a user by their normalised address.
func (r *UserRepository) ByEmail(ctx context.Context, email entity.Email) (*entity.User, error) {
	row, err := queries(ctx, r.pool).GetUserByEmail(ctx, email.String())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("persistence - ByEmail: %w", entity.ErrUserNotFound)
		}

		return nil, fmt.Errorf("persistence - ByEmail - GetUserByEmail: %w", err)
	}

	user, err := toUserEntity(&row)
	if err != nil {
		return nil, fmt.Errorf("persistence - ByEmail: %w", err)
	}

	return user, nil
}

// toUserEntity rebuilds the aggregate from a row. The stored email and role are
// re-parsed rather than cast, so a row that was written around the application
// cannot smuggle an unknown role into a live aggregate.
func toUserEntity(row *sqlcgen.User) (*entity.User, error) {
	email, err := entity.ParseEmail(row.Email)
	if err != nil {
		return nil, fmt.Errorf("toUserEntity: %w", err)
	}

	role, err := entity.ParseRole(row.Role)
	if err != nil {
		return nil, fmt.Errorf("toUserEntity: %w", err)
	}

	return entity.ReconstituteUser(
		row.ID,
		email,
		row.Name,
		row.PasswordHash,
		role,
		row.Version,
		row.CreatedAt,
		row.UpdatedAt,
	), nil
}
