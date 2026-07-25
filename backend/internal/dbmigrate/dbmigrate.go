// Package dbmigrate applies the embedded schema migrations at process start.
//
// Why in-process instead of a separate `migrate` step: the production target is
// a single container with no init/job phase, and it may run more than one
// replica. golang-migrate takes a Postgres advisory lock, so if two replicas
// boot at once exactly one migrates and the other waits and finds nothing to do.
package dbmigrate

import (
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // registers the "pgx5" scheme
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"sabai-pos/backend/migrations"
)

// Up applies every pending migration. It is a no-op when the schema is current.
// Returns the version the database ended up at, and whether anything changed.
func Up(databaseURL string) (version uint, changed bool, err error) {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return 0, false, fmt.Errorf("read embedded migrations: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, pgxURL(databaseURL))
	if err != nil {
		return 0, false, fmt.Errorf("open migrator: %w", err)
	}
	// Close reports both the source and database close errors; neither is fatal
	// to a successful migration, so they are only surfaced when Up succeeded.
	defer func() {
		if srcErr, dbErr := m.Close(); err == nil {
			err = errors.Join(srcErr, dbErr)
		}
	}()

	changed = true
	if upErr := m.Up(); upErr != nil {
		if !errors.Is(upErr, migrate.ErrNoChange) {
			return 0, false, fmt.Errorf("apply migrations: %w", upErr)
		}
		changed = false
	}

	version, dirty, verErr := m.Version()
	if verErr != nil {
		return 0, changed, fmt.Errorf("read schema version: %w", verErr)
	}
	if dirty {
		return version, changed, fmt.Errorf("schema version %d is dirty — a previous migration failed halfway and needs manual repair", version)
	}
	return version, changed, nil
}

// pgxURL rewrites a standard Postgres URL onto the scheme golang-migrate's
// pgx/v5 driver registers, so the app and the migrator share one DATABASE_URL.
func pgxURL(u string) string {
	for _, scheme := range []string{"postgres://", "postgresql://"} {
		if strings.HasPrefix(u, scheme) {
			return "pgx5://" + strings.TrimPrefix(u, scheme)
		}
	}
	return u
}
