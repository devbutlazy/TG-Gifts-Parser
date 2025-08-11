package external

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"tg-gifts-parser/internal/parser"

	_ "github.com/mattn/go-sqlite3"
)

const (
	dbFolder        = "data/database"
	giftsJSONPath   = "data/gifts.json"
	updateThreshold = 10000
)

func ensureGiftsTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS gifts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		number INTEGER,
		model TEXT,
		backdrop TEXT,
		symbol TEXT
	);`
	_, err := db.Exec(query)
	if err != nil {
		fmt.Printf("Error creating gifts table: %v\n", err)
	}
	return err
}

func ensureColumns(db *sql.DB) error {
	requiredCols := map[string]string{
		"name":     "TEXT",
		"number":   "INTEGER",
		"model":    "TEXT",
		"backdrop": "TEXT",
		"symbol":   "TEXT",
	}

	rows, err := db.Query("PRAGMA table_info(gifts)")
	if err != nil {
		fmt.Printf("Error querying table info: %v\n", err)
		return err
	}
	defer rows.Close()

	existingCols := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, dflt_value, pk interface{}
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt_value, &pk); err != nil {
			fmt.Printf("Error scanning table info: %v\n", err)
			return err
		}
		existingCols[name] = true
	}

	for col, colType := range requiredCols {
		if !existingCols[col] {
			alterStmt := fmt.Sprintf("ALTER TABLE gifts ADD COLUMN %s %s;", col, colType)
			if _, err := db.Exec(alterStmt); err != nil {
				fmt.Printf("Error adding column %s: %v\n", col, err)
				return err
			}
			fmt.Printf("Added missing column %s to gifts table\n", col)
		}
	}
	return nil
}

func getExistingCount(dbPath string) (int, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		fmt.Printf("Failed to open db %s: %v\n", dbPath, err)
		return 0, err
	}
	defer db.Close()

	if err := ensureGiftsTable(db); err != nil {
		return 0, err
	}
	if err := ensureColumns(db); err != nil {
		return 0, err
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM gifts").Scan(&count)
	if err != nil {
		fmt.Printf("Failed to count rows in db %s: %v\n", dbPath, err)
		return 0, err
	}
	return count, nil
}

func updateGiftIfNeeded(key string) (int, error) {
	keySlug := parser.SanitizeKey(key)
	dbPath := filepath.Join(dbFolder, keySlug+".db")

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fmt.Printf("DB for %q does not exist, parsing from start\n", key)
		parser.ParseAndSaveGift(key, nil, nil)
		return 0, nil
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open db for %q: %w", key, err)
	}
	defer db.Close()

	if err := ensureGiftsTable(db); err != nil {
		return 0, fmt.Errorf("failed to ensure gifts table for %q: %w", key, err)
	}
	if err := ensureColumns(db); err != nil {
		return 0, fmt.Errorf("failed to ensure columns for %q: %w", key, err)
	}

	existingCount, err := getExistingCount(dbPath)
	if err != nil {
		return 0, fmt.Errorf("failed to get existing count for %q: %w", key, err)
	}

	url := fmt.Sprintf("https://t.me/nft/%s-1", keySlug)
	doc, err := parser.FetchHTML(url, 3, 2*time.Second)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch first page for %q: %w", key, err)
	}

	quantityStr := parser.ExtractGiftField(doc, "Quantity")
	if quantityStr == "Unknown" {
		quantityStr = parser.ExtractQuantityFallback(doc)
	}
	quantity := parser.CleanQuantity(quantityStr)
	if quantity == 0 {
		fmt.Printf("Zero quantity found for %q\n", key)
		return 0, nil
	}

	if existingCount >= quantity {
		return 0, nil
	}

	newItemsCount := 0
	for i := existingCount + 1; i <= quantity; i++ {
		giftURL := fmt.Sprintf("https://t.me/nft/%s-%d", keySlug, i)
		doc, err := parser.FetchHTML(giftURL, 3, 2*time.Second)
		if err != nil {
			fmt.Printf("Warning: failed to fetch %s: %v\n", giftURL, err)
			continue
		}

		info := parser.ParseGiftInfo(doc)
		err = parser.InsertGift(db, key, info["Model"], info["Backdrop"], info["Symbol"], i)
		if err != nil {
			fmt.Printf("Error inserting gift %q #%d: %v\n", key, i, err)
			continue
		}

		newItemsCount++
		if newItemsCount%1000 == 0 {
			fmt.Printf("Parsed %q gift item #%d\n", key, i)
		}
	}

	fmt.Printf("Updated gift %q with %d new items\n", key, newItemsCount)
	return newItemsCount, nil
}

func RunUpdater() (int, error) {
	keys, err := parser.LoadGiftsJSON(giftsJSONPath)
	if err != nil {
		return 0, fmt.Errorf("failed to load gifts JSON: %w", err)
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)
	totalNewItems := 0

	for _, key := range keys {
		wg.Add(1)
		sem <- struct{}{}
		go func(k string) {
			defer wg.Done()
			defer func() { <-sem }()
			count, err := updateGiftIfNeeded(k)
			if err != nil {
				fmt.Printf("Update error for %q: %v\n", k, err)
				return
			}
			mu.Lock()
			totalNewItems += count
			mu.Unlock()
		}(key)
	}
	wg.Wait()

	fmt.Printf("Total new gifts added this run: %d\n", totalNewItems)
	return totalNewItems, nil
}

func gitCommitAll(newItems int) error {
	cmd := exec.Command("git", "add", dbFolder)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	commitMsg := fmt.Sprintf("chore(data/database/*.db): updated %d gifts", newItems)
	cmd = exec.Command("git", "commit", "-m", commitMsg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}

	cmd = exec.Command("git", "push")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git push failed: %w", err)
	}

	return nil
}

func ScheduleUpdater() {
	totalSinceLastCommit := 0

	for {
		fmt.Println("Running updater...")
		newItems, err := RunUpdater()
		if err != nil {
			fmt.Printf("Updater error: %v\n", err)
		} else {
			totalSinceLastCommit += newItems
			if totalSinceLastCommit >= updateThreshold {
				fmt.Println("Threshold reached, committing changes to GitHub...")
				err := gitCommitAll(totalSinceLastCommit)
				if err != nil {
					fmt.Printf("Git commit failed: %v\n", err)
				} else {
					totalSinceLastCommit = 0
				}
			}
		}

		fmt.Println("Sleeping for 6 hours...")
		time.Sleep(6 * time.Hour)
	}
}
