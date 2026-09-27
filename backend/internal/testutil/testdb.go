package testutil

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/cageymage/fuzion/backend/migrations"
)

var sharedDB *sqlx.DB

// Run starts one migrated Postgres container for the whole test package and
// tears it down afterwards; DB hands each test a clean set of tables from it.
func Run(m *testing.M) int {
	ctx := context.Background()

	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("fuzion_test"),
		postgres.WithUsername("fuzion"),
		postgres.WithPassword("fuzion"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(2*time.Minute),
		),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start postgres container: %v\n", err)
		return 1
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "terminate postgres container: %v\n", err)
		}
	}()

	databaseURL, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "read postgres connection string: %v\n", err)
		return 1
	}

	if err := migrations.Apply(databaseURL); err != nil {
		fmt.Fprintf(os.Stderr, "migrate test database: %v\n", err)
		return 1
	}

	db, err := sqlx.ConnectContext(ctx, "pgx", databaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect to test database: %v\n", err)
		return 1
	}
	defer db.Close()

	sharedDB = db
	return m.Run()
}

func DB(t *testing.T) *sqlx.DB {
	t.Helper()

	if sharedDB == nil {
		t.Fatal("testutil.Run must wrap TestMain before a test asks for the database")
	}
	if _, err := sharedDB.Exec(`TRUNCATE applications, news_posts, raids, streams, sessions, users, professions, characters`); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
	return sharedDB
}
