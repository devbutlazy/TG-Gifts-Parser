package utils

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func QueryEntries(dbPath, model, backdrop, symbol string) ([]int, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query("SELECT number, model, backdrop, symbol FROM gifts")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []int
	for rows.Next() {
		var number int
		var dbModel, dbBackdrop, dbSymbol string
		err := rows.Scan(&number, &dbModel, &dbBackdrop, &dbSymbol)
		if err != nil {
			return nil, err
		}

		dbModelClean := RemovePercent(dbModel)
		dbBackdropClean := RemovePercent(dbBackdrop)
		dbSymbolClean := RemovePercent(dbSymbol)

		modelMatch := dbModelClean == model
		backdropMatch := backdrop == "" || dbBackdropClean == backdrop
		symbolMatch := symbol == "" || dbSymbolClean == symbol

		if modelMatch && backdropMatch && symbolMatch {
			matches = append(matches, number)
		}
	}

	return matches, nil
}
