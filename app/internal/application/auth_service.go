package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/kethaka-creskit/go-ddd-service/internal/domain/auth/entity"
)

// AuthService exposes the identity use cases: registering, signing in and out,
// resolving a session token back to the user who holds it.
type AuthService struct {
	users    entity.UserRepository
	sessions entity.SessionRepository
	tx       TxManager
	clock    Clock
	ids      IDGenerator
	hasher   PasswordHasher
	tokens   TokenGenerator
	ttl      time.Duration
	log      *slog.Logger

	// decoy is a valid digest of a password nobody has. Verifying against it on
	// an unknown email makes a failed login cost the same as a successful one,
	// so the response time does not reveal which addresses are registered. It is
	// computed once, on first use, because hashing is deliberately slow.
	decoy func() string
}

// NewAuthService wires the service. Every dependency is an interface owned by
// the domain or by this package, so nothing here depends on Postgres or HTTP.
func NewAuthService(
	users entity.UserRepository,
	sessions entity.SessionRepository,
	tx TxManager,
	clock Clock,
	ids IDGenerator,
	hasher PasswordHasher,
	tokens TokenGenerator,
	ttl SessionTTL,
	log *slog.Logger,
) *AuthService {
	return &AuthService{
		users:    users,
		sessions: sessions,
		tx:       tx,
		clock:    clock,
		ids:      ids,
		hasher:   hasher,
		tokens:   tokens,
		ttl:      time.Duration(ttl),
		log:      log,
		decoy: sync.OnceValue(func() string {
			// A failure here only costs the timing defence, and there is nobody to
			// report it to from inside a lazily evaluated closure. Verify against
			// the empty string will simply fail, which is the required outcome.
			hash, _ := hasher.Hash(uuid.NewString())

			return hash
		}),
	}
}

// Register creates a user and signs them in.
//
// The uniqueness check and the insert share one transaction, and the unique
// index on email is what actually enforces uniqueness: the lookup exists to
// produce a clean ErrEmailTaken for the ordinary case, and the constraint
// catches the two-registrations-at-once race the lookup cannot see.
func (s *AuthService) Register(
	ctx context.Context,
	email, name, password string,
) (*entity.User, entity.Credential, error) {
	if err := entity.ValidatePassword(password); err != nil {
		return nil, entity.Credential{}, fmt.Errorf("application - Register - ValidatePassword: %w", err)
	}

	addr, err := entity.ParseEmail(email)
	if err != nil {
		return nil, entity.Credential{}, fmt.Errorf("application - Register - ParseEmail: %w", err)
	}

	hash, err := s.hasher.Hash(password)
	if err != nil {
		return nil, entity.Credential{}, fmt.Errorf("application - Register - hasher.Hash: %w", err)
	}

	var (
		user *entity.User
		cred entity.Credential
	)

	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		switch _, err := s.users.ByEmail(ctx, addr); {
		case err == nil:
			return entity.ErrEmailTaken
		case !errors.Is(err, entity.ErrUserNotFound):
			return fmt.Errorf("users.ByEmail: %w", err)
		}

		registered, err := entity.Register(
			s.ids.NewID(), addr.String(), name, hash, entity.RoleMember, s.clock.Now(),
		)
		if err != nil {
			return fmt.Errorf("entity.Register: %w", err)
		}

		if err := s.users.Add(ctx, registered); err != nil {
			return fmt.Errorf("users.Add: %w", err)
		}

		issued, err := s.issueSession(ctx, registered.ID())
		if err != nil {
			return err
		}

		// Publish to the enclosing scope only once everything has succeeded, so a
		// rolled-back attempt cannot leave a half-built user visible to the caller.
		user, cred = registered, issued

		return nil
	})
	if err != nil {
		return nil, entity.Credential{}, fmt.Errorf("application - Register: %w", err)
	}

	s.log.InfoContext(ctx, "user registered",
		slog.String("user_id", user.ID().String()),
	)

	return user, cred, nil
}

// Login verifies a password and starts a session.
//
// Every failure returns entity.ErrInvalidCredentials, whether the address is
// unregistered, malformed, or the password is simply wrong. Telling the caller
// which one it was turns the login form into an account-enumeration oracle.
func (s *AuthService) Login(ctx context.Context, email, password string) (*entity.User, entity.Credential, error) {
	addr, err := entity.ParseEmail(email)
	if err != nil {
		return nil, entity.Credential{}, fmt.Errorf("application - Login - ParseEmail: %w", entity.ErrInvalidCredentials)
	}

	user, err := s.users.ByEmail(ctx, addr)
	if err != nil {
		if errors.Is(err, entity.ErrUserNotFound) {
			// Spend the same time as a real verification would.
			_ = s.hasher.Verify(s.decoy(), password)

			return nil, entity.Credential{}, fmt.Errorf("application - Login: %w", entity.ErrInvalidCredentials)
		}

		return nil, entity.Credential{}, fmt.Errorf("application - Login - users.ByEmail: %w", err)
	}

	if err := s.hasher.Verify(user.PasswordHash(), password); err != nil {
		if errors.Is(err, ErrPasswordMismatch) {
			return nil, entity.Credential{}, fmt.Errorf("application - Login: %w", entity.ErrInvalidCredentials)
		}

		return nil, entity.Credential{}, fmt.Errorf("application - Login - hasher.Verify: %w", err)
	}

	cred, err := s.issueSession(ctx, user.ID())
	if err != nil {
		return nil, entity.Credential{}, fmt.Errorf("application - Login: %w", err)
	}

	s.log.InfoContext(ctx, "user signed in",
		slog.String("user_id", user.ID().String()),
	)

	return user, cred, nil
}

// Authenticate resolves a session token to the user holding it. This is the
// call the REST middleware makes on every protected request.
//
// An expired session is deleted on the way past rather than left to the
// sweeper: the request already paid for the lookup.
func (s *AuthService) Authenticate(ctx context.Context, token string) (*entity.User, error) {
	if token == "" {
		return nil, fmt.Errorf("application - Authenticate: %w", entity.ErrSessionNotFound)
	}

	now := s.clock.Now()

	session, err := s.sessions.ByTokenHash(ctx, s.tokens.Hash(token), now)
	if err != nil {
		return nil, fmt.Errorf("application - Authenticate - sessions.ByTokenHash: %w", err)
	}

	if session.IsExpired(now) {
		if err := s.sessions.Delete(ctx, session.ID()); err != nil {
			s.log.WarnContext(ctx, "failed to delete an expired session",
				slog.String("session_id", session.ID().String()),
				slog.Any("error", err),
			)
		}

		return nil, fmt.Errorf("application - Authenticate: %w", entity.ErrSessionExpired)
	}

	user, err := s.users.ByID(ctx, session.UserID())
	if err != nil {
		return nil, fmt.Errorf("application - Authenticate - users.ByID: %w", err)
	}

	return user, nil
}

// Logout ends one session. It is idempotent: signing out with a token that is
// already gone is a success, because the caller's desired state holds.
func (s *AuthService) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}

	session, err := s.sessions.ByTokenHash(ctx, s.tokens.Hash(token), s.clock.Now())
	if err != nil {
		if errors.Is(err, entity.ErrSessionNotFound) {
			return nil
		}

		return fmt.Errorf("application - Logout - sessions.ByTokenHash: %w", err)
	}

	if err := s.sessions.Delete(ctx, session.ID()); err != nil {
		return fmt.Errorf("application - Logout - sessions.Delete: %w", err)
	}

	s.log.InfoContext(ctx, "user signed out",
		slog.String("user_id", session.UserID().String()),
	)

	return nil
}

// CurrentUser loads a user by id, for the handler that has already been
// authenticated by the middleware.
func (s *AuthService) CurrentUser(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	user, err := s.users.ByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("application - CurrentUser - users.ByID: %w", err)
	}

	return user, nil
}

// Rename changes a user's display name.
func (s *AuthService) Rename(ctx context.Context, id uuid.UUID, name string) (*entity.User, error) {
	var user *entity.User

	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		found, err := s.users.ByID(ctx, id)
		if err != nil {
			return fmt.Errorf("users.ByID: %w", err)
		}

		if err := found.Rename(name, s.clock.Now()); err != nil {
			return fmt.Errorf("user.Rename: %w", err)
		}

		if err := s.users.Update(ctx, found); err != nil {
			return fmt.Errorf("users.Update: %w", err)
		}

		user = found

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("application - Rename: %w", err)
	}

	return user, nil
}

// ChangePassword replaces a user's password and re-issues their session.
//
// Every other session for the user is destroyed. A password change is how
// somebody evicts an attacker who already has a valid cookie, and it only does
// that if the old cookies stop working; the caller gets a fresh token so the
// browser doing the change is not signed out along with the attacker.
func (s *AuthService) ChangePassword(
	ctx context.Context,
	id uuid.UUID,
	current, next string,
) (*entity.User, entity.Credential, error) {
	if err := entity.ValidatePassword(next); err != nil {
		return nil, entity.Credential{}, fmt.Errorf("application - ChangePassword - ValidatePassword: %w", err)
	}

	var (
		user *entity.User
		cred entity.Credential
	)

	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		found, err := s.users.ByID(ctx, id)
		if err != nil {
			return fmt.Errorf("users.ByID: %w", err)
		}

		if err := s.hasher.Verify(found.PasswordHash(), current); err != nil {
			if errors.Is(err, ErrPasswordMismatch) {
				return entity.ErrInvalidCredentials
			}

			return fmt.Errorf("hasher.Verify: %w", err)
		}

		hash, err := s.hasher.Hash(next)
		if err != nil {
			return fmt.Errorf("hasher.Hash: %w", err)
		}

		if err := found.ChangePassword(hash, s.clock.Now()); err != nil {
			return fmt.Errorf("user.ChangePassword: %w", err)
		}

		if err := s.users.Update(ctx, found); err != nil {
			return fmt.Errorf("users.Update: %w", err)
		}

		// Order matters: wipe every session first, then mint the replacement, or
		// the caller's brand-new session would be deleted along with the rest.
		if err := s.sessions.DeleteByUser(ctx, found.ID()); err != nil {
			return fmt.Errorf("sessions.DeleteByUser: %w", err)
		}

		issued, err := s.issueSession(ctx, found.ID())
		if err != nil {
			return err
		}

		user, cred = found, issued

		return nil
	})
	if err != nil {
		return nil, entity.Credential{}, fmt.Errorf("application - ChangePassword: %w", err)
	}

	s.log.InfoContext(ctx, "password changed",
		slog.String("user_id", user.ID().String()),
	)

	return user, cred, nil
}

// PurgeExpiredSessions deletes sessions that are past their expiry. Nothing
// depends on it for correctness -- an expired session never authenticates
// anyone -- so it exists only to keep the table from growing without bound.
func (s *AuthService) PurgeExpiredSessions(ctx context.Context) (int64, error) {
	deleted, err := s.sessions.DeleteExpired(ctx, s.clock.Now())
	if err != nil {
		return 0, fmt.Errorf("application - PurgeExpiredSessions - sessions.DeleteExpired: %w", err)
	}

	return deleted, nil
}

// issueSession mints a token and stores its digest. The plaintext is returned
// to the caller and deliberately never written anywhere else.
func (s *AuthService) issueSession(ctx context.Context, userID uuid.UUID) (entity.Credential, error) {
	token, err := s.tokens.New()
	if err != nil {
		return entity.Credential{}, fmt.Errorf("tokens.New: %w", err)
	}

	session, err := entity.IssueSession(s.ids.NewID(), userID, s.tokens.Hash(token), s.ttl, s.clock.Now())
	if err != nil {
		return entity.Credential{}, fmt.Errorf("entity.IssueSession: %w", err)
	}

	if err := s.sessions.Add(ctx, session); err != nil {
		return entity.Credential{}, fmt.Errorf("sessions.Add: %w", err)
	}

	return entity.Credential{Session: session, Token: token}, nil
}
