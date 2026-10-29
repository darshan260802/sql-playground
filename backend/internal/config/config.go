// Package config reads settings from environment variables.
package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Listen string // HTTP listen address

	MySQLAddr          string // host:port
	MySQLAdminUser     string // account that can create schemas, users and grants
	MySQLAdminPassword string

	AppSchema   string // application tables (users, sessions, saved queries, ...)
	CoreSchema  string // the pristine dataset everyone starts from
	UserPrefix  string // per-user schemas are <prefix><id>
	SpareCopies int    // pre-built copies of core kept ready for instant signup/reset

	MaxUsers         int
	UserQuotaMB      int
	QueryTimeout     time.Duration // logged-in users
	AnonQueryTimeout time.Duration // anonymous read-only queries
	MaxRows          int           // rows returned to the browser per result set
	MaxConcurrent    int           // queries executing at once across all users
	SessionIdle      time.Duration // idle time before a user's SQL session is closed

	StaticDir    string // built frontend; empty disables static serving
	CookieSecure bool
	SessionDays  int
	TrustProxy   bool // read client IP from X-Forwarded-For / X-Real-IP (behind nginx)
}

func Load() Config {
	return Config{
		Listen:             env("SQLPG_LISTEN", ":8080"),
		MySQLAddr:          env("SQLPG_MYSQL_ADDR", "127.0.0.1:3306"),
		MySQLAdminUser:     env("SQLPG_MYSQL_ADMIN_USER", "root"),
		MySQLAdminPassword: env("SQLPG_MYSQL_ADMIN_PASSWORD", ""),
		AppSchema:          env("SQLPG_APP_SCHEMA", "playground_app"),
		CoreSchema:         env("SQLPG_CORE_SCHEMA", "core"),
		UserPrefix:         env("SQLPG_USER_SCHEMA_PREFIX", "u_"),
		SpareCopies:        envInt("SQLPG_SPARE_COPIES", 1),
		MaxUsers:           envInt("SQLPG_MAX_USERS", 20),
		UserQuotaMB:        envInt("SQLPG_USER_QUOTA_MB", 300),
		QueryTimeout:       envDur("SQLPG_QUERY_TIMEOUT", 30*time.Second),
		AnonQueryTimeout:   envDur("SQLPG_ANON_QUERY_TIMEOUT", 15*time.Second),
		MaxRows:            envInt("SQLPG_MAX_ROWS", 2000),
		MaxConcurrent:      envInt("SQLPG_MAX_CONCURRENT", 8),
		SessionIdle:        envDur("SQLPG_SESSION_IDLE", 15*time.Minute),
		StaticDir:          env("SQLPG_STATIC_DIR", ""),
		CookieSecure:       envBool("SQLPG_COOKIE_SECURE", false),
		SessionDays:        envInt("SQLPG_SESSION_DAYS", 30),
		TrustProxy:         envBool("SQLPG_TRUST_PROXY", false),
	}
}

func env(k, def string) string {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(k)); err == nil {
		return v
	}
	return def
}

func envBool(k string, def bool) bool {
	if v, err := strconv.ParseBool(os.Getenv(k)); err == nil {
		return v
	}
	return def
}

func envDur(k string, def time.Duration) time.Duration {
	if v, err := time.ParseDuration(os.Getenv(k)); err == nil {
		return v
	}
	return def
}
