// Package config loads all configuration once, at startup, into one value.
// Nothing else in the module reads the environment.
package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config is the whole configuration surface of the service. env-required makes
// startup fail loudly on a missing value rather than defaulting silently.
type Config struct {
	App     App
	HTTP    HTTP
	Log     Log
	DB      DB
	Session Session
}

// App identifies the running service.
type App struct {
	Name string `env:"APP_NAME" env-default:"go-ddd-service"`
	Env  string `env:"APP_ENV"  env-default:"development"`
}

// HTTP configures the public listener.
type HTTP struct {
	Port            string        `env:"HTTP_PORT"             env-default:"8080"`
	ReadTimeout     time.Duration `env:"HTTP_READ_TIMEOUT"     env-default:"5s"`
	WriteTimeout    time.Duration `env:"HTTP_WRITE_TIMEOUT"    env-default:"10s"`
	IdleTimeout     time.Duration `env:"HTTP_IDLE_TIMEOUT"     env-default:"60s"`
	HandlerTimeout  time.Duration `env:"HTTP_HANDLER_TIMEOUT"  env-default:"15s"`
	ShutdownTimeout time.Duration `env:"HTTP_SHUTDOWN_TIMEOUT" env-default:"10s"`
}

// Log configures structured logging.
type Log struct {
	Level  string `env:"LOG_LEVEL"  env-default:"info"`
	Format string `env:"LOG_FORMAT" env-default:"json"`
}

// DB configures the Postgres connection pool.
type DB struct {
	URL             string        `env:"DATABASE_URL" env-required:"true"`
	MaxConns        int32         `env:"DB_MAX_CONNS"         env-default:"10"`
	MinConns        int32         `env:"DB_MIN_CONNS"         env-default:"2"`
	MaxConnLifetime time.Duration `env:"DB_MAX_CONN_LIFETIME" env-default:"1h"`
	MaxConnIdleTime time.Duration `env:"DB_MAX_CONN_IDLE"     env-default:"30m"`
	ConnectTimeout  time.Duration `env:"DB_CONNECT_TIMEOUT"   env-default:"5s"`
	ConnectAttempts int           `env:"DB_CONNECT_ATTEMPTS"  env-default:"10"`
}

// Session configures the login session and the cookie that carries it.
type Session struct {
	// CookieName is the cookie the session token travels in. It is short and
	// meaningless on purpose: a cookie called "session_token" advertises itself.
	CookieName string `env:"SESSION_COOKIE_NAME" env-default:"sid"`
	// TTL is how long a session stays valid after it is issued.
	TTL time.Duration `env:"SESSION_TTL" env-default:"720h"`
	// CookieDomain scopes the cookie. Empty means host-only, which is the safer
	// default: set it only when the API and the UI live on sibling subdomains.
	CookieDomain string `env:"SESSION_COOKIE_DOMAIN" env-default:""`
	// CookieSecure restricts the cookie to HTTPS. It is forced on in production
	// regardless of this value; the setting exists so a staging deployment on
	// plain HTTP can be told to relax it, not so production can be.
	CookieSecure bool `env:"SESSION_COOKIE_SECURE" env-default:"false"`
}

// SecureCookies reports whether the session cookie must be HTTPS-only.
//
// Production is not allowed to opt out. A Secure-less session cookie is
// readable by anything on the path the first time a user types the bare
// hostname, and no configuration mistake should be able to arrange that.
func (c *Config) SecureCookies() bool {
	return c.Session.CookieSecure || c.IsProduction()
}

// Load reads configuration from the process environment.
//
// Precedence is environment first: an .env file is a local-development
// convenience only, loaded by the Makefile, never by the running service. That
// keeps the twelve-factor contract intact in every deployed environment.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := cleanenv.ReadEnv(cfg); err != nil {
		return nil, fmt.Errorf("config - Load: %w (see .env.example for the full list)", err)
	}

	return cfg, nil
}

// LoadFromFile reads configuration from an env file, then overlays the
// environment. Intended for tests and local runs.
func LoadFromFile(path string) (*Config, error) {
	cfg := &Config{}
	if err := cleanenv.ReadConfig(path, cfg); err != nil {
		return nil, fmt.Errorf("config - LoadFromFile(%s): %w", path, err)
	}

	return cfg, nil
}

// IsProduction reports whether the service believes it is in production.
func (c *Config) IsProduction() bool { return c.App.Env == "production" }
