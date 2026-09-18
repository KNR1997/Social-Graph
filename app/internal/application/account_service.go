package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/kethaka-creskit/go-ddd-service/internal/domain/account/entity"
)

// AccountService exposes the account use cases.
type AccountService struct {
	accounts entity.Repository
	tx       TxManager
	clock    Clock
	ids      IDGenerator
	log      *slog.Logger
}

// NewAccountService wires the service. Every dependency is an interface owned
// by the domain or by this package, so nothing here depends on Postgres or HTTP.
func NewAccountService(
	accounts entity.Repository,
	tx TxManager,
	clock Clock,
	ids IDGenerator,
	log *slog.Logger,
) *AccountService {
	return &AccountService{
		accounts: accounts,
		tx:       tx,
		clock:    clock,
		ids:      ids,
		log:      log,
	}
}

// OpenAccount creates a new account.
func (s *AccountService) OpenAccount(ctx context.Context, owner, currency string) (*entity.Account, error) {
	acc, err := entity.Open(s.ids.NewID(), owner, currency, s.clock.Now())
	if err != nil {
		return nil, fmt.Errorf("application - OpenAccount - entity.Open: %w", err)
	}

	if err := s.accounts.Add(ctx, acc); err != nil {
		return nil, fmt.Errorf("application - OpenAccount - accounts.Add: %w", err)
	}

	s.log.InfoContext(ctx, "account opened",
		slog.String("account_id", acc.ID().String()),
		slog.String("currency", string(acc.Balance().Currency())),
	)

	return acc, nil
}

// GetAccount loads a single account.
func (s *AccountService) GetAccount(ctx context.Context, id uuid.UUID) (*entity.Account, error) {
	acc, err := s.accounts.ByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("application - GetAccount - accounts.ByID: %w", err)
	}

	return acc, nil
}

// ListAccounts returns a page of accounts.
func (s *AccountService) ListAccounts(ctx context.Context, page entity.Page) ([]*entity.Account, error) {
	accounts, err := s.accounts.List(ctx, page)
	if err != nil {
		return nil, fmt.Errorf("application - ListAccounts - accounts.List: %w", err)
	}

	return accounts, nil
}

// Deposit credits an account. The read and the write share one transaction, so
// a concurrent writer either loses the optimistic-lock race or serialises
// behind this one; the balance can never be computed from a stale read.
func (s *AccountService) Deposit(
	ctx context.Context,
	id uuid.UUID,
	minor int64,
	currency string,
) (*entity.Account, error) {
	acc, err := s.mutate(ctx, id, func(acc *entity.Account, amount entity.Money) error {
		return acc.Deposit(amount, s.clock.Now())
	}, minor, currency)
	if err != nil {
		return nil, fmt.Errorf("application - Deposit: %w", err)
	}

	return acc, nil
}

// Withdraw debits an account.
func (s *AccountService) Withdraw(
	ctx context.Context,
	id uuid.UUID,
	minor int64,
	currency string,
) (*entity.Account, error) {
	acc, err := s.mutate(ctx, id, func(acc *entity.Account, amount entity.Money) error {
		return acc.Withdraw(amount, s.clock.Now())
	}, minor, currency)
	if err != nil {
		return nil, fmt.Errorf("application - Withdraw: %w", err)
	}

	return acc, nil
}

// CloseAccount closes an account.
func (s *AccountService) CloseAccount(ctx context.Context, id uuid.UUID) (*entity.Account, error) {
	var acc *entity.Account

	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		loaded, err := s.accounts.ByID(ctx, id)
		if err != nil {
			return fmt.Errorf("accounts.ByID: %w", err)
		}
		if err := loaded.Close(s.clock.Now()); err != nil {
			return err
		}
		if err := s.accounts.Update(ctx, loaded); err != nil {
			return fmt.Errorf("accounts.Update: %w", err)
		}

		acc = loaded

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("application - CloseAccount: %w", err)
	}

	return acc, nil
}

// mutate is the load-change-store skeleton shared by the balance use cases.
// Note how little it does: parse the input into a value object, call one method
// on the aggregate, persist. All the rules are on the other side of that call.
func (s *AccountService) mutate(
	ctx context.Context,
	id uuid.UUID,
	change func(*entity.Account, entity.Money) error,
	minor int64,
	currency string,
) (*entity.Account, error) {
	amount, err := entity.NewMoney(minor, currency)
	if err != nil {
		return nil, err
	}

	var acc *entity.Account

	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		loaded, err := s.accounts.ByID(ctx, id)
		if err != nil {
			return fmt.Errorf("accounts.ByID: %w", err)
		}
		if err := change(loaded, amount); err != nil {
			return err
		}
		if err := s.accounts.Update(ctx, loaded); err != nil {
			return fmt.Errorf("accounts.Update: %w", err)
		}

		acc = loaded

		return nil
	})
	if err != nil {
		return nil, err
	}

	return acc, nil
}
