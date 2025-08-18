package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	_ "github.com/marcboeker/go-duckdb"
)

func main() {
	inputDir := "data/database"
	outputDir := "data/database-duckdb"

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatal(err)
	}

	files, err := filepath.Glob(filepath.Join(inputDir, "*.db"))
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		fmt.Println("Processing:", f)

		sqliteDB, err := sql.Open("sqlite3", f)
		if err != nil {
			log.Fatal(err)
		}

		// список таблиць
		rows, err := sqliteDB.Query(`SELECT name FROM sqlite_master WHERE type='table'`)
		if err != nil {
			log.Fatal(err)
		}

		var tables []string
		for rows.Next() {
			var t string
			if err := rows.Scan(&t); err != nil {
				log.Fatal(err)
			}
			if t != "sqlite_sequence" {
				tables = append(tables, t)
			}
		}
		rows.Close()

		sqliteDB.Close()

		// DuckDB (in-memory)
		duck, err := sql.Open("duckdb", "")
		if err != nil {
			log.Fatal(err)
		}

		outFile := filepath.Join(outputDir, strings.TrimSuffix(filepath.Base(f), ".db")+".parquet")

		for _, table := range tables {
			// читаємо таблицю в DuckDB через ATTACH sqlite
			query := fmt.Sprintf("INSTALL sqlite; LOAD sqlite; ATTACH '%s' AS db (TYPE SQLITE);", f)
			if _, err := duck.Exec(query); err != nil {
				log.Fatalf("attach sqlite failed: %v", err)
			}

			copyQuery := fmt.Sprintf("COPY (SELECT * FROM db.%s) TO '%s' (FORMAT PARQUET);", table, outFile)
			if _, err := duck.Exec(copyQuery); err != nil {
				log.Fatalf("export table %s failed: %v", table, err)
			}
		}

		duck.Close()
		fmt.Println("Saved:", outFile)
	}
}

