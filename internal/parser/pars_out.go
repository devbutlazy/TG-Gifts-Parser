package parser

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/antchfx/htmlquery"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/net/html"
)

const workerCount = 10

type GiftItem struct {
	Name     string
	Model    string
	Backdrop string
	Symbol   string
	Number   int
}

func ExtractQuantityFallback(doc *html.Node) string {
	text := htmlquery.InnerText(doc)
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Quantity") {
			fields := strings.Fields(line)
			for _, f := range fields {
				fClean := strings.Trim(f, ":,() ")
				if n, err := strconv.Atoi(fClean); err == nil && n > 0 {
					return fClean
				}
			}
		}
	}
	return "Unknown"
}

func SanitizeKey(key string) string {
	var b strings.Builder
	for _, r := range key {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func LoadGiftsJSON(path string) ([]string, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read gifts.json: %w", err)
	}

	var gifts map[string]interface{}
	if err := json.Unmarshal(file, &gifts); err != nil {
		return nil, fmt.Errorf("failed to parse gifts.json: %w", err)
	}

	keys := make([]string, 0, len(gifts))
	for k := range gifts {
		keys = append(keys, k)
	}
	return keys, nil
}

func CreateDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS gifts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		model TEXT,
		backdrop TEXT,
		symbol TEXT,
		number INTEGER
	);
	CREATE INDEX IF NOT EXISTS idx_name_number ON gifts(name, number);
	`
	if _, err := db.Exec(createTableSQL); err != nil {
		return nil, fmt.Errorf("failed to create table/index: %w", err)
	}
	return db, nil
}

func InsertGift(db *sql.DB, name, model, backdrop, symbol string, number int) error {
	_, err := db.Exec(
		`INSERT INTO gifts (name, model, backdrop, symbol, number) VALUES (?, ?, ?, ?, ?)`,
		name, model, backdrop, symbol, number,
	)
	return err
}

func ParseAndSaveGift(key string, wg *sync.WaitGroup, sem chan struct{}) {
	defer wg.Done()

	keySlug := SanitizeKey(key)
	dbPath := filepath.Join("data/database", keySlug+".db")

	fmt.Printf("Starting parsing gift %q, db file: %s\n", key, dbPath)

	db, err := CreateDB(dbPath)
	if err != nil {
		fmt.Printf("Failed to create DB for %q: %v\n", key, err)
		return
	}
	defer db.Close()

	url := fmt.Sprintf("https://t.me/nft/%s-1", keySlug)
	doc, err := FetchHTML(url, 3, 2*time.Second)
	if err != nil {
		fmt.Printf("Failed to fetch first page for %q: %v\n", key, err)
		return
	}

	quantityStr := ExtractGiftField(doc, "Quantity")
	if quantityStr == "Unknown" {
		quantityStr = ExtractQuantityFallback(doc)
	}
	quantity := CleanQuantity(quantityStr)
	if quantity == 0 {
		fmt.Printf("Warning: zero quantity for %q\n", key)
		return
	}

	fmt.Printf("Gift %q has quantity %d\n", key, quantity)

	for i := 1; i <= quantity; i++ {
		giftURL := fmt.Sprintf("https://t.me/nft/%s-%d", keySlug, i)
		doc, err := FetchHTML(giftURL, 3, 2*time.Second)
		if err != nil {
			fmt.Printf("Warning: failed to fetch %s: %v\n", giftURL, err)
			continue
		}

		info := ParseGiftInfo(doc)

		err = InsertGift(db, key, info["Model"], info["Backdrop"], info["Symbol"], i)
		if err != nil {
			fmt.Printf("Error inserting gift %q #%d: %v\n", key, i, err)
		}

		if i%1000 == 0 {
			fmt.Printf("Parsed %q gift item #%d\n", key, i)
		}
	}

	fmt.Printf("Finished parsing gift %q\n", key)
	<-sem
}

func ParseAllGifts() error {
	keys, err := LoadGiftsJSON(filepath.Join("data", "gifts.json"))
	if err != nil {
		return err
	}

	if err := os.MkdirAll("data/database", 0755); err != nil {
		return fmt.Errorf("failed to create database folder: %w", err)
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, workerCount)

	fmt.Printf("Starting to parse %d gifts concurrently with %d workers...\n", len(keys), workerCount)

	for _, key := range keys {
		wg.Add(1)
		sem <- struct{}{}
		go ParseAndSaveGift(key, &wg, sem)
	}

	wg.Wait()
	fmt.Println("All gifts parsed and saved to separate .db files in the 'data/database/' folder")
	return nil
}
