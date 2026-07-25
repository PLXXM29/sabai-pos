// Command seed populates a database with the sample store, staff accounts and
// catalogue. It is the first-install step for a self-hosted deployment and the
// convenience path for local development; the public demo seeds itself on boot.
//
// By default it does nothing to a database that already has a store, so it is
// safe to leave in a provisioning script:
//
//	DATABASE_URL=postgres://... go run ./cmd/seed
//	DATABASE_URL=postgres://... go run ./cmd/seed -history 30   # + a month of sample sales
//	DATABASE_URL=postgres://... go run ./cmd/seed -reset        # wipe first (destructive)
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	_ "time/tzdata" // sample sales are timestamped in Asia/Bangkok

	"github.com/jackc/pgx/v5/pgxpool"

	"sabai-pos/backend/internal/demo"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "seed error:", err)
		os.Exit(1)
	}
}

func run() error {
	history := flag.Int("history", 0,
		"days of generated sample sales (0 = catalogue and staff only)")
	reset := flag.Bool("reset", false,
		"DESTRUCTIVE: erase all existing business data first")
	flag.Parse()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL is required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()

	seeder := demo.New(pool)

	if *reset {
		res, err := seeder.Reset(ctx, *history)
		if err != nil {
			return err
		}
		report("reset", res)
		return nil
	}

	res, seeded, err := seeder.Ensure(ctx, *history)
	if err != nil {
		return err
	}
	if !seeded {
		fmt.Println("seed skipped: this database already has a store (use -reset to rebuild)")
		return nil
	}
	report("seeded", res)
	return nil
}

func report(verb string, res demo.Result) {
	fmt.Printf("%s %q · users %d · products %d · bills %d over %d days · %s\n",
		verb, res.ShopName, res.Users, res.Products, res.Bills, res.HistoryDays, res.Took)
	fmt.Println("logins:")
	for _, a := range demo.Accounts {
		fmt.Printf("  %-8s / %-12s %s\n", a.Username, a.Password, a.Role)
	}
}
