package config

import "testing"

func TestNormalizeDBURL(t *testing.T) {
	cases := []struct{ in, want string }{
		// Legacy JDBC URL is rewritten to a pgx postgres:// URL.
		{"jdbc:postgresql://localhost:5432/unigate", "postgres://localhost:5432/unigate"},
		// currentSchema (isolated test schema) must survive the rewrite.
		{"jdbc:postgresql://db:5432/unigate?currentSchema=isolated", "postgres://db:5432/unigate?currentSchema=isolated"},
		{"jdbc:postgresql://db:5432/unigate?currentSchema=isolated&sslmode=disable", "postgres://db:5432/unigate?currentSchema=isolated&sslmode=disable"},
		{"jdbc:postgresql://localhost:5432/unigate?currentSchema=test%20schema", "postgres://localhost:5432/unigate?currentSchema=test%20schema"},
		// Already a postgres URL: userinfo and query are preserved.
		{"postgres://u:p@host:5432/db?sslmode=disable", "postgres://u:p@host:5432/db?sslmode=disable"},
		{"postgresql://host/db", "postgresql://host/db"},
		// keyword/value DSN passes through untouched.
		{"host=localhost dbname=unigate user=unigate", "host=localhost dbname=unigate user=unigate"},
		{"host=localhost dbname=unigate options=-c search_path=isolated", "host=localhost dbname=unigate options=-c search_path=isolated"},
		// Surrounding whitespace is trimmed.
		{"  jdbc:postgresql://localhost/db  ", "postgres://localhost/db"},
	}
	for _, tc := range cases {
		got, err := NormalizeDBURL(tc.in)
		if err != nil {
			t.Errorf("NormalizeDBURL(%q) error: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("NormalizeDBURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeDBURLInvalid(t *testing.T) {
	for _, in := range []string{"postgres://host/%zz", "jdbc:postgresql://host/%zz"} {
		if _, err := NormalizeDBURL(in); err == nil {
			t.Errorf("NormalizeDBURL(%q) accepted invalid URL", in)
		}
	}
}
