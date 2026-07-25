// Package migrations exposes the .sql migration files as an embedded
// filesystem so the API binary can migrate itself on boot.
//
// The files stay plain golang-migrate files in this directory: the `migrate`
// CLI (docker-compose dev flow) and the embedded runner read exactly the same
// source, so there is no second definition of the schema to drift.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
