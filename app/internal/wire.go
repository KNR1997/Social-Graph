//go:build wireinject
// +build wireinject

// Package internal holds the dependency-injection graph. This file is the only
// place where interfaces are bound to concrete implementations, so the direction
// of every dependency in the service is auditable by reading it.
package internal

import (
	"context"

	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kethaka-creskit/go-ddd-service/config"
	"github.com/kethaka-creskit/go-ddd-service/internal/application"
	accountentity "github.com/kethaka-creskit/go-ddd-service/internal/domain/account/entity"
	authentity "github.com/kethaka-creskit/go-ddd-service/internal/domain/auth/entity"
	personentity "github.com/kethaka-creskit/go-ddd-service/internal/domain/person/entity"
	postentity "github.com/kethaka-creskit/go-ddd-service/internal/domain/post/entity"
	"github.com/kethaka-creskit/go-ddd-service/internal/infrastructure/persistence"
	"github.com/kethaka-creskit/go-ddd-service/internal/infrastructure/security"
	"github.com/kethaka-creskit/go-ddd-service/internal/infrastructure/system"
	"github.com/kethaka-creskit/go-ddd-service/internal/interfaces/rest"
	accountrest "github.com/kethaka-creskit/go-ddd-service/internal/interfaces/rest/account"
	authrest "github.com/kethaka-creskit/go-ddd-service/internal/interfaces/rest/auth"
	personrest "github.com/kethaka-creskit/go-ddd-service/internal/interfaces/rest/person"
	postrest "github.com/kethaka-creskit/go-ddd-service/internal/interfaces/rest/post"
	"github.com/kethaka-creskit/go-ddd-service/pkg/httpserver"
	"github.com/kethaka-creskit/go-ddd-service/pkg/logger"
	"github.com/kethaka-creskit/go-ddd-service/pkg/postgres"
)

// coreSet is everything above the database: it takes a *pgxpool.Pool as given.
// Splitting the graph here is what lets the integration tests reuse the exact
// production wiring against a throwaway container pool, instead of maintaining
// a second, subtly different graph.
var coreSet = wire.NewSet(
	logger.New,

	persistence.NewAccountRepository,
	persistence.NewPersonRepository,
	persistence.NewPostRepository,
	persistence.NewUserRepository,
	persistence.NewSessionRepository,
	persistence.NewTxManager,
	system.NewClock,
	system.NewIDGenerator,
	security.NewHasher,
	security.NewTokenGenerator,

	// Configuration is unpacked into the narrow values the layers below actually
	// consume, so neither the application service nor the handler has to be
	// handed the whole *config.Config to reach one field.
	provideSessionTTL,
	provideCookieConfig,

	application.NewAccountService,
	application.NewPersonService,
	application.NewPostService,
	application.NewAuthService,

	accountrest.NewHandler,
	personrest.NewHandler,
	postrest.NewHandler,
	authrest.NewHandler,
	rest.NewRouter,
	httpserver.New,

	wire.Struct(new(API), "*"),

	// The bindings. Each one points from an inner-layer interface to its
	// outer-layer implementation, which is the dependency inversion made literal.
	wire.Bind(new(accountentity.Repository), new(*persistence.AccountRepository)),
	wire.Bind(new(personentity.Repository), new(*persistence.PersonRepository)),
	wire.Bind(new(postentity.Repository), new(*persistence.PostRepository)),
	wire.Bind(new(application.TxManager), new(*persistence.TxManager)),
	wire.Bind(new(application.Clock), new(system.Clock)),
	wire.Bind(new(application.IDGenerator), new(system.IDGenerator)),
	wire.Bind(new(accountrest.Service), new(*application.AccountService)),
	wire.Bind(new(personrest.Service), new(*application.PersonService)),
	wire.Bind(new(postrest.Service), new(*application.PostService)),
	wire.Bind(new(authentity.UserRepository), new(*persistence.UserRepository)),
	wire.Bind(new(authentity.SessionRepository), new(*persistence.SessionRepository)),
	wire.Bind(new(application.PasswordHasher), new(*security.Hasher)),
	wire.Bind(new(application.TokenGenerator), new(security.TokenGenerator)),
	wire.Bind(new(authrest.Service), new(*application.AuthService)),
)

// InitializeAPI builds the whole service, opening its own pool.
func InitializeAPI(ctx context.Context, cfg *config.Config) (*API, error) {
	wire.Build(coreSet, postgres.New, wire.Bind(new(rest.Pinger), new(*pgxpool.Pool)))

	return nil, nil
}

// InitializeAPIWithPool builds the service on a caller-supplied pool.
func InitializeAPIWithPool(cfg *config.Config, pool *pgxpool.Pool) *API {
	wire.Build(coreSet, wire.Bind(new(rest.Pinger), new(*pgxpool.Pool)))

	return nil
}
