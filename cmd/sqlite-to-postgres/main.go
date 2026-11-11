// Command sqlite-to-postgres copies every row from the old SQLite database
// into an empty PostgreSQL database, in one transaction: either all of it
// is copied or none of it is.
//
//	go run ./cmd/sqlite-to-postgres -from data/progress.db -to "$DATABASE_URL"
//
// Stop the server first, so nothing is written to SQLite while it copies.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	_ "modernc.org/sqlite"

	"crud_app/internal/infrastructure/postgres"
)

func main() {
	from := flag.String("from", "data/progress.db", "path of the SQLite database to copy from")
	to := flag.String("to", os.Getenv("DATABASE_URL"), "PostgreSQL URL to copy into (default $DATABASE_URL)")
	flag.Parse()
	if *to == "" {
		log.Fatal("no PostgreSQL URL: pass -to or set DATABASE_URL")
	}
	if _, err := os.Stat(*from); err != nil {
		log.Fatalf("SQLite database: %v", err)
	}

	src, err := sql.Open("sqlite", "file:"+*from+"?mode=ro&_pragma=busy_timeout(5000)")
	if err != nil {
		log.Fatal(err)
	}
	defer src.Close()

	// Open creates the tables if they're missing.
	dst, err := postgres.Open(*to)
	if err != nil {
		log.Fatal(err)
	}
	defer dst.Close()

	if err := copyAll(context.Background(), src, dst); err != nil {
		log.Fatalf("nothing copied: %v", err)
	}
	fmt.Println("done")
}

func copyAll(ctx context.Context, src, dst *sql.DB) error {
	tx, err := dst.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, table := range postgres.Tables {
		var n int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			return fmt.Errorf("table %s in PostgreSQL already has %d rows; copy only into an empty database", table, n)
		}
	}
	for _, table := range postgres.Tables {
		n, err := copyTable(ctx, src, tx, table)
		if err != nil {
			return fmt.Errorf("%s: %w", table, err)
		}
		fmt.Printf("%-16s %d rows\n", table, n)
	}
	return tx.Commit()
}

// copyTable copies the columns the PostgreSQL table has from the SQLite
// table of the same name.
func copyTable(ctx context.Context, src *sql.DB, tx *sql.Tx, table string) (int, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT column_name, data_type FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = $1
		ORDER BY ordinal_position`, table)
	if err != nil {
		return 0, err
	}
	var columns []string
	var boolean []bool
	for rows.Next() {
		var name, typ string
		if err := rows.Scan(&name, &typ); err != nil {
			rows.Close()
			return 0, err
		}
		columns = append(columns, name)
		boolean = append(boolean, typ == "boolean")
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	list := strings.Join(columns, ", ")
	placeholders := make([]string, len(columns))
	for i := range columns {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	insert, err := tx.PrepareContext(ctx,
		`INSERT INTO `+table+` (`+list+`) VALUES (`+strings.Join(placeholders, ", ")+`)`)
	if err != nil {
		return 0, err
	}
	defer insert.Close()

	from, err := src.QueryContext(ctx, `SELECT `+list+` FROM `+table)
	if err != nil {
		return 0, err
	}
	defer from.Close()

	copied := 0
	values := make([]any, len(columns))
	ptrs := make([]any, len(columns))
	for i := range values {
		ptrs[i] = &values[i]
	}
	for from.Next() {
		if err := from.Scan(ptrs...); err != nil {
			return 0, err
		}
		// SQLite stored these flags as 0 and 1
		for i, isBool := range boolean {
			if n, ok := values[i].(int64); isBool && ok {
				values[i] = n != 0
			}
		}
		if _, err := insert.ExecContext(ctx, values...); err != nil {
			return 0, err
		}
		copied++
	}
	return copied, from.Err()
}
