// Package config loads deployment configuration from environment variables.
// It mirrors the Java application.yml property bindings (ADR-048 keeps the
// same environment names so existing deployments keep working).
package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// Config holds all runtime settings. Values come from the environment only;
// no credentials are baked into the binary.
type Config struct {
	Port               int
	DBURL              string // PostgreSQL DSN (pgx format)
	DBUser             string
	DBPassword         string
	MasterKey          string // Base64Std 32 bytes
	CookieSecure       bool
	AllowedOrigin      string
	BootstrapUsername  string
	BootstrapPassword  string
	KeyDirectory       string
	StorageDirectory   string
	ResetPasswordFor   string // one-shot non-web password reset mode
	ResetMode          bool   // true when --reset-password-for / env reset mode requested
	PublicBaseURL      string
	Notifications      Notifications
	Providers          Providers
	SourcesPollMS      int
	LifecyclePollMS    int
	NotificationsPollMS int
}

// Notifications mirrors unigate.notifications.* and spring.mail.* settings.
type Notifications struct {
	EmailEnabled   bool
	SMSEnabled     bool
	SMSEndpoint    string
	SMSAllowedHosts []string
	SMSToken       string
	From           string
	AllowLocalTest bool
	MailHost       string
	MailPort       int
	MailUsername   string
	MailPassword   string
	SMTPAuth       bool
	SMTPStartTLS   bool
}

// Providers mirrors unigate.providers.connections.<id>.<name> settings.
// The raw environment is kept so per-connection settings resolve lazily,
// exactly like Spring Environment.getProperty in the Java implementation.
type Providers struct {
	Raw map[string]string
}

// Setting returns unigate.providers.connections.<connectionID>.<name> or "".
func (p Providers) Setting(connectionID, name string) string {
	return p.Raw["unigate.providers.connections."+connectionID+"."+name]
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func getbool(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	b, err := strconv.ParseBool(strings.TrimSpace(v))
	if err != nil {
		return fallback
	}
	return b
}

func getint(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return fallback
	}
	return n
}

// Load reads configuration and validates it fail-fast.
func Load() (*Config, error) {
	cfg := &Config{
		Port:              getint("PORT", 8080),
		DBUser:            getenv("UNIGATE_DB_USER", "unigate"),
		DBPassword:        os.Getenv("UNIGATE_DB_PASSWORD"),
		MasterKey:         os.Getenv("UNIGATE_MASTER_KEY"),
		CookieSecure:      getbool("UNIGATE_COOKIE_SECURE", true),
		AllowedOrigin:     getenv("UNIGATE_ALLOWED_ORIGIN", "http://localhost:5173"),
		BootstrapUsername: getenv("UNIGATE_BOOTSTRAP_USERNAME", "admin"),
		BootstrapPassword: os.Getenv("UNIGATE_BOOTSTRAP_PASSWORD"),
		KeyDirectory:      getenv("UNIGATE_KEY_DIRECTORY", "./var/keys"),
		StorageDirectory:  getenv("UNIGATE_STORAGE_DIRECTORY", "./var/unigate"),
		ResetPasswordFor:  os.Getenv("UNIGATE_RESET_PASSWORD_FOR"),
		PublicBaseURL:     getenv("UNIGATE_PUBLIC_BASE_URL", getenv("UNIGATE_ALLOWED_ORIGIN", "http://localhost:5173")),
		SourcesPollMS:       getint("UNIGATE_SOURCES_POLL_MS", 3000),
		LifecyclePollMS:     getint("UNIGATE_LIFECYCLE_POLL_MS", 60000),
		NotificationsPollMS: getint("UNIGATE_NOTIFICATIONS_POLL_MS", 3000),
		Notifications: Notifications{
			EmailEnabled:    getbool("UNIGATE_NOTIFICATIONS_EMAIL_ENABLED", false),
			SMSEnabled:      getbool("UNIGATE_NOTIFICATIONS_SMS_ENABLED", false),
			SMSEndpoint:     os.Getenv("UNIGATE_NOTIFICATIONS_SMS_ENDPOINT"),
			SMSToken:        os.Getenv("UNIGATE_NOTIFICATIONS_SMS_TOKEN"),
			From:            os.Getenv("UNIGATE_NOTIFICATIONS_FROM"),
			AllowLocalTest:  getbool("UNIGATE_NOTIFICATIONS_ALLOW_LOCAL_TEST", false),
			MailHost:        os.Getenv("SPRING_MAIL_HOST"),
			MailPort:        getint("SPRING_MAIL_PORT", 25),
			MailUsername:    os.Getenv("SPRING_MAIL_USERNAME"),
			MailPassword:    os.Getenv("SPRING_MAIL_PASSWORD"),
			SMTPAuth:        getbool("SPRING_MAIL_PROPERTIES_MAIL_SMTP_AUTH", false),
			SMTPStartTLS:    getbool("SPRING_MAIL_PROPERTIES_MAIL_SMTP_STARTTLS_ENABLE", true),
		},
	}
	for _, h := range strings.Split(os.Getenv("UNIGATE_NOTIFICATIONS_SMS_ALLOWED_HOSTS"), ",") {
		cfg.Notifications.SMSAllowedHosts = append(cfg.Notifications.SMSAllowedHosts, strings.TrimSpace(h))
	}
	// Reset mode: explicit flag, or the legacy env var combined with --no-web
	// (the equivalent of Spring's web-application-type=none in reset-admin.sh).
	noWeb := false
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if v, ok := strings.CutPrefix(arg, "--reset-password-for="); ok {
			cfg.ResetPasswordFor = v
			cfg.ResetMode = true
		} else if arg == "--reset-password-for" && i+1 < len(args) {
			cfg.ResetPasswordFor = args[i+1]
			cfg.ResetMode = true
			i++
		} else if arg == "--no-web" || arg == "--reset" {
			noWeb = true
		}
	}
	if cfg.ResetPasswordFor != "" && noWeb {
		cfg.ResetMode = true
	}
	if cfg.ResetPasswordFor == "" {
		cfg.ResetMode = false
	}
	raw := getenv("UNIGATE_DB_URL", "jdbc:postgresql://localhost:5432/unigate")
	dsn, err := NormalizeDBURL(raw)
	if err != nil {
		return nil, fmt.Errorf("parse UNIGATE_DB_URL: %w", err)
	}
	cfg.DBURL = dsn
	if cfg.Port < 1 || cfg.Port > 65535 {
		return nil, fmt.Errorf("PORT must be between 1 and 65535")
	}
	penv := map[string]string{}
	for _, kv := range os.Environ() {
		k, v, _ := strings.Cut(kv, "=")
		penv[k] = v
	}
	cfg.Providers = Providers{Raw: penv}
	return cfg, nil
}

// NormalizeDBURL accepts both the legacy JDBC URL and a PostgreSQL DSN/URL and
// returns a pgx-compatible connection string. Query parameters such as
// currentSchema (used by the isolated test schema) are preserved.
func NormalizeDBURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "jdbc:postgresql://") {
		raw = "postgres://" + strings.TrimPrefix(raw, "jdbc:postgresql://")
	}
	if strings.HasPrefix(raw, "postgres://") || strings.HasPrefix(raw, "postgresql://") {
		u, err := url.Parse(raw)
		if err != nil {
			return "", fmt.Errorf("invalid database URL: %w", err)
		}
		return u.String(), nil
	}
	// Already a keyword/value DSN (host=... dbname=...).
	return raw, nil
}