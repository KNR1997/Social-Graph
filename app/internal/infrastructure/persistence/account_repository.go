package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kethaka-creskit/go-ddd-service/internal/domain/account/entity"
	"github.com/kethaka-creskit/go-ddd-service/internal/infrastructure/persistence/sqlcgen"
)

// uniqueViolation is the Postgres SQLSTATE for a duplicate key.
const uniqueViolation = "23505"

// AccountRepository implements entity.Repository.
type AccountRepository struct {
	pool *pgxpool.Pool
}

// Compile-time proof that this adapter satisfies the domain port. Cheap, and it
// fails the build rather than the wiring when the interface changes.
var _ entity.Repository = (*AccountRepository)(nil)

// NewAccountRepository builds the repository.
func NewAccountRepository(pool *pgxpool.Pool) *AccountRepository {
	return &AccountRepository{pool: pool}
}

// Add inserts a newly opened account.
func (r *AccountRepository) Add(ctx context.Context, a *entity.Account) error {
	err := queries(ctx, r.pool).InsertAccount(ctx, sqlcgen.InsertAccountParams{
		ID:           a.ID(),
		Owner:        a.Owner(),
		BalanceMinor: a.Balance().Minor(),
		Currency:     string(a.Balance().Currency()),
		Status:       string(a.Status()),
		Version:      a.Version(),
		CreatedAt:    a.CreatedAt(),
		UpdatedAt:    a.UpdatedAt(),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return fmt.Errorf("persistence - Add: %w", entity.ErrConflict)
		}

		return fmt.Errorf("persistence - Add - InsertAccount: %w", err)
	}

	return nil
}

// Update writes a changed account, guarded by the version it was loaded at.
func (r *AccountRepository) Update(ctx context.Context, a *entity.Account) error {
	rows, err := queries(ctx, r.pool).UpdateAccount(ctx, sqlcgen.UpdateAccountParams{
		ID:           a.ID(),
		BalanceMinor: a.Balance().Minor(),
		Status:       string(a.Status()),
		UpdatedAt:    a.UpdatedAt(),
		Version:      a.Version(),
	})
	if err != nil {
		return fmt.Errorf("persistence - Update - UpdateAccount: %w", err)
	}

	// Zero rows means either the id is gone or another writer moved the version
	// on. Both are conflicts from this caller's point of view.
	if rows == 0 {
		return fmt.Errorf("persistence - Update: %w", entity.ErrConflict)
	}

	return nil
}

// ByID loads one account.
func (r *AccountRepository) ByID(ctx context.Context, id uuid.UUID) (*entity.Account, error) {
	row, err := queries(ctx, r.pool).GetAccount(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("persistence - ByID: %w", entity.ErrNotFound)
		}

		return nil, fmt.Errorf("persistence - ByID - GetAccount: %w", err)
	}

	acc, err := toEntity(&row)
	if err != nil {
		return nil, fmt.Errorf("persistence - ByID: %w", err)
	}

	return acc, nil
}

// List returns a page of accounts, newest first.
func (r *AccountRepository) List(ctx context.Context, page entity.Page) ([]*entity.Account, error) {
	rows, err := queries(ctx, r.pool).ListAccounts(ctx, sqlcgen.ListAccountsParams{
		Limit:  page.Limit,
		Offset: page.Offset,
	})
	if err != nil {
		return nil, fmt.Errorf("persistence - List - ListAccounts: %w", err)
	}

	accounts := make([]*entity.Account, 0, len(rows))
	// Index rather than range-copy: the generated row struct is large enough that
	// copying it per iteration shows up in profiles on wide pages.
	for i := range rows {
		acc, err := toEntity(&rows[i])
		if err != nil {
			return nil, fmt.Errorf("persistence - List: %w", err)
		}

		accounts = append(accounts, acc)
	}

	return accounts, nil
}

// toEntity maps a row to the aggregate. This is the anti-corruption seam: the
// column layout can change without the domain noticing, and a row that cannot
// be mapped is a hard error rather than a half-built Account.
func toEntity(row *sqlcgen.Account) (*entity.Account, error) {
	balance, err := entity.NewMoney(row.BalanceMinor, row.Currency)
	if err != nil {
		return nil, fmt.Errorf("account %s: %w", row.ID, err)
	}

	status, err := entity.ParseStatus(row.Status)
	if err != nil {
		return nil, fmt.Errorf("account %s: %w", row.ID, err)
	}

	return entity.Reconstitute(
		row.ID,
		row.Owner,
		balance,
		status,
		row.Version,
		row.CreatedAt,
		row.UpdatedAt,
	), nil
}
