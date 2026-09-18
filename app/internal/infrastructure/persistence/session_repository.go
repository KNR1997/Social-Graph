package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kethaka-creskit/go-ddd-service/internal/domain/auth/entity"
	"github.com/kethaka-creskit/go-ddd-service/internal/infrastructure/persistence/sqlcgen"
)

// SessionRepository is the Postgres adapter for entity.SessionRepository.
type SessionRepository struct {
	pool *pgxpool.Pool
}

var _ entity.SessionRepository = (*SessionRepository)(nil)

// NewSessionRepository builds the repository.
func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{pool: pool}
}

// Add stores a newly issued session.
func (r *SessionRepository) Add(ctx context.Context, s *entity.Session) error {
	err := queries(ctx, r.pool).InsertSession(ctx, sqlcgen.InsertSessionParams{
		ID:        s.ID(),
		UserID:    s.UserID(),
		TokenHash: s.TokenHash(),
		IssuedAt:  s.IssuedAt(),
		ExpiresAt: s.ExpiresAt(),
	})
	if err != nil {
		return fmt.Errorf("persistence - Add - InsertSession: %w", err)
	}

	return nil
}

// ByTokenHash loads a session by its stored digest, excluding expired rows.
//
// An expired session and a nonexistent one are the same answer on purpose:
// ErrSessionNotFound. The caller is a middleware deciding whether to let a
// request through, and both mean no.
func (r *SessionRepository) ByTokenHash(
	ctx context.Context,
	tokenHash string,
	now time.Time,
) (*entity.Session, error) {
	row, err := queries(ctx, r.pool).GetSessionByTokenHash(ctx, sqlcgen.GetSessionByTokenHashParams{
		TokenHash: tokenHash,
		ExpiresAt: now,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("persistence - ByTokenHash: %w", entity.ErrSessionNotFound)
		}

		return nil, fmt.Errorf("persistence - ByTokenHash - GetSessionByTokenHash: %w", err)
	}

	return entity.ReconstituteSession(
		row.ID,
		row.UserID,
		row.TokenHash,
		row.IssuedAt,
		row.ExpiresAt,
	), nil
}

// Delete removes one session. Deleting a row that is already gone is not an
// error: sign-out is idempotent.
func (r *SessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := queries(ctx, r.pool).DeleteSession(ctx, id); err != nil {
		return fmt.Errorf("persistence - Delete - DeleteSession: %w", err)
	}

	return nil
}

// DeleteByUser removes every session belonging to a user.
func (r *SessionRepository) DeleteByUser(ctx context.Context, userID uuid.UUID) error {
	if err := queries(ctx, r.pool).DeleteSessionsByUser(ctx, userID); err != nil {
		return fmt.Errorf("persistence - DeleteByUser - DeleteSessionsByUser: %w", err)
	}

	return nil
}

// DeleteExpired sweeps sessions that are past their expiry and reports how many
// went.
func (r *SessionRepository) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	rows, err := queries(ctx, r.pool).DeleteExpiredSessions(ctx, now)
	if err != nil {
		return 0, fmt.Errorf("persistence - DeleteExpired - DeleteExpiredSessions: %w", err)
	}

	return rows, nil
}
