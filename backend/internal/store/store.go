package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB bundles the generated Queries with the connection pool and adds
// transaction support. Services depend on *DB (or the Querier interface for
// mocking in unit tests). Named DB — not Store — because sqlc already generates
// a Store model from the `stores` table.
type DB struct {
	*Queries
	pool *pgxpool.Pool
}

func NewDB(pool *pgxpool.Pool) *DB {
	return &DB{Queries: New(pool), pool: pool}
}

func (s *DB) Pool() *pgxpool.Pool { return s.pool }

// ExecTx runs fn inside a single database transaction. Any error (or panic)
// rolls the whole thing back — this is how a sale stays atomic
// (bill + items + stock in one commit).
func (s *DB) ExecTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()
	if err := fn(New(tx)); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}

// ── pgtype conversion helpers ────────────────────────────────────────────────

func PgUUID(u uuid.UUID) pgtype.UUID { return pgtype.UUID{Bytes: u, Valid: true} }

func PgUUIDPtr(u *uuid.UUID) pgtype.UUID {
	if u == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *u, Valid: true}
}

func PgTime(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }
