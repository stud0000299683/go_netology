package db

import "testing"

func TestDSNPrefersDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@db:5432/cashcraft?sslmode=disable")
	if got := DSN(); got != "postgres://user:pass@db:5432/cashcraft?sslmode=disable" {
		t.Fatalf("DSN = %s", got)
	}
}

func TestDSNFromSeparateVariables(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_HOST", "db.local")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_USER", "cashier")
	t.Setenv("DB_PASS", "secret")
	t.Setenv("DB_NAME", "cashapp")

	const want = "postgres://cashier:secret@db.local:5433/cashapp?sslmode=disable"
	if got := DSN(); got != want {
		t.Fatalf("DSN = %s", got)
	}
}
