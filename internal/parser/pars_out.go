package parser

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"net/http"
	"strconv"

	"github.com/antchfx/htmlquery"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/net/html"
)

func FetchHTML(url string) (*html.Node, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	doc, err := htmlquery.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}
	return doc, nil
}

func extractGiftField(doc *html.Node, field string) string {
	td := htmlquery.FindOne(doc, fmt.Sprintf(`//th[text()="%s"]/following-sibling::td`, field))
	if td == nil {
		return "Unknown"
	}

	text := strings.TrimSpace(htmlquery.InnerText(td))
	mark := htmlquery.FindOne(td, "mark")
	if mark != nil {
		return fmt.Sprintf("%s (%s)", strings.TrimSpace(htmlquery.InnerText(td)), strings.TrimSpace(htmlquery.InnerText(mark)))
	}
	return text
}

func ParseGiftInfo(doc *html.Node) map[string]string {
	info := make(map[string]string)

	if ownerTd := htmlquery.FindOne(doc, `//th[text()="Owner"]/following-sibling::td`); ownerTd != nil {
		var name, href string
		if a := htmlquery.FindOne(ownerTd, `a`); a != nil {
			href = strings.TrimSpace(htmlquery.SelectAttr(a, "href"))
			if span := htmlquery.FindOne(a, `span`); span != nil {
				name = htmlquery.InnerText(span)
			} else {
				name = htmlquery.InnerText(a)
			}
		} else if span := htmlquery.FindOne(ownerTd, `span`); span != nil {
			name = htmlquery.InnerText(span)
		} else {
			name = htmlquery.InnerText(ownerTd)
		}

		info["Owner"] = strings.TrimSpace(name)
		if href != "" {
			info["Owner"] += fmt.Sprintf(" (%s)", href)
		}
	} else {
		info["Owner"] = "Unknown"
	}

	for _, field := range []string{"Model", "Backdrop", "Symbol", "Quantity"} {
		info[field] = extractGiftField(doc, field)
	}

	return info
}

func CleanQuantity(q string) int {
	cleaned := strings.Split(q, `/`)
	num, _ := strconv.Atoi(strings.ReplaceAll(strings.ReplaceAll(cleaned[0], "\u00A0", ""), " ", ""))
	return num
}

const workerCount = 20
const insertBatchSize = 5000

type GiftItem struct {
	Name     string
	Model    string
	Backdrop string
	Symbol   string
	Number   int
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

func batchInsertGifts(db *sql.DB, giftChan <-chan GiftItem) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`INSERT INTO gifts (name, model, backdrop, symbol, number) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	count := 0
	start := time.Now()

	for gift := range giftChan {
		if _, err := stmt.Exec(gift.Name, gift.Model, gift.Backdrop, gift.Symbol, gift.Number); err != nil {
			fmt.Printf("Error inserting gift %s #%d: %v\n", gift.Name, gift.Number, err)
		}
		count++

		if count%insertBatchSize == 0 {
			if err := tx.Commit(); err != nil {
				return err
			}
			duration := time.Since(start)
			fmt.Printf("Inserted %d gifts in %v\n", count, duration)

			tx, err = db.Begin()
			if err != nil {
				return err
			}
			stmt, err = tx.Prepare(`INSERT INTO gifts (name, model, backdrop, symbol, number) VALUES (?, ?, ?, ?, ?)`)
			if err != nil {
				return err
			}
			start = time.Now()
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	fmt.Printf("Total inserted gifts: %d\n", count)
	return nil
}

func ParseAllGifts() error {
	fmt.Println("Loading gift keys from data/gifts.json...")
	keys, err := LoadGiftsJSON(filepath.Join("data", "gifts.json"))
	if err != nil {
		return err
	}
	fmt.Printf("Loaded %d gift keys\n", len(keys))

	db, err := CreateDB("database.db")
	if err != nil {
		return err
	}
	defer db.Close()

	giftChan := make(chan GiftItem, 10000)
	var wg sync.WaitGroup
	sem := make(chan struct{}, workerCount)

	dbDone := make(chan struct{})
	go func() {
		defer close(dbDone)
		if err := batchInsertGifts(db, giftChan); err != nil {
			fmt.Printf("DB insert error: %v\n", err)
		}
	}()

	totalGifts := 0

	for _, key := range keys {
		keySlug := strings.ReplaceAll(key, " ", "-")
		url := fmt.Sprintf("https://t.me/nft/%s-1", keySlug)

		fmt.Printf("Parsing gift %q first page to get quantity: %s\n", key, url)
		doc, err := FetchHTML(url)
		if err != nil {
			fmt.Printf("Warning: failed to fetch %s: %v\n", url, err)
			continue
		}

		quantityStr := extractGiftField(doc, "Quantity")
		quantity := CleanQuantity(quantityStr)
		if quantity == 0 {
			fmt.Printf("Warning: zero quantity for %s\n", key)
			continue
		}
		fmt.Printf("Gift %q has quantity %d\n", key, quantity)
		totalGifts += quantity

		for i := 1; i <= quantity; i++ {
			wg.Add(1)
			sem <- struct{}{}

			go func(name, slug string, number int) {
				defer wg.Done()
				defer func() { <-sem }()

				giftURL := fmt.Sprintf("https://t.me/nft/%s-%d", slug, number)
				doc, err := FetchHTML(giftURL)
				if err != nil {
					fmt.Printf("Warning: failed to fetch %s: %v\n", giftURL, err)
					return
				}

				info := ParseGiftInfo(doc)

				giftChan <- GiftItem{
					Name:     name,
					Model:    info["Model"],
					Backdrop: info["Backdrop"],
					Symbol:   info["Symbol"],
					Number:   number,
				}

				if number%10000 == 0 {
					fmt.Printf("Parsed %q gift item #%d\n", name, number)
				}
			}(key, keySlug, i)
		}
	}

	fmt.Printf("Started parsing a total of %d gifts...\n", totalGifts)
	wg.Wait()
	close(giftChan)
	<-dbDone

	fmt.Println("All gifts parsed and saved to database.db")
	return nil
}
