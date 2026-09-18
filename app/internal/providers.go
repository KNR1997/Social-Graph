package internal

import (
	"github.com/kethaka-creskit/go-ddd-service/config"
	"github.com/kethaka-creskit/go-ddd-service/internal/application"
	authrest "github.com/kethaka-creskit/go-ddd-service/internal/interfaces/rest/auth"
)

// This file holds the handful of providers that exist only to narrow
// *config.Config down to the specific values a layer consumes. They live beside
// the graph rather than inside wire.go because wire.go is build-tagged for the
// generator and never compiled into the binary.

// provideSessionTTL narrows the config down to the one duration the application
// layer needs, so that layer keeps depending on a value rather than on config.
func provideSessionTTL(cfg *config.Config) application.SessionTTL {
	return application.SessionTTL(cfg.Session.TTL)
}

// provideCookieConfig builds the transport-level cookie policy.
//
// Secure comes from cfg.SecureCookies rather than straight from the field, so
// the production override lives in exactly one place and no injector can route
// around it.
func provideCookieConfig(cfg *config.Config) authrest.CookieConfig {
	return authrest.CookieConfig{
		Name:   cfg.Session.CookieName,
		Domain: cfg.Session.CookieDomain,
		Secure: cfg.SecureCookies(),
	}
}
