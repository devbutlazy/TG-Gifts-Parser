package updater

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"tg-gifts-parser/internal/parser" 
)

const workerCount = 10

func getLastParsedIndex(dbPath string) int {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return 0
	}
	defer db.Close()

	var max int
	row := db.QueryRow("SELECT MAX(number) FROM gifts")
	if err := row.Scan(&max); err != nil {
		return 0
	}
	return max
}

func updateGift(key string, wg *sync.WaitGroup, sem chan struct{}) {
	defer wg.Done()

	keySlug := parser.SanitizeKey(key)
	dbPath := filepath.Join("database", keySlug+".db")
	last := getLastParsedIndex(dbPath)

	url := fmt.Sprintf("https://t.me/nft/%s-1", keySlug)
	doc, err := parser.FetchHTML(url, 3, 2*time.Second)
	if err != nil {
		fmt.Printf("Failed to fetch base URL for %q: %v\n", key, err)
		<-sem
		return
	}

	quantityStr := parser.ExtractGiftField(doc, "Quantity")
	if quantityStr == "Unknown" {
		quantityStr = parser.ExtractQuantityFallback(doc)
	}
	quantity := parser.CleanQuantity(quantityStr)

	if quantity <= last {
		fmt.Printf("Gift %q is up-to-date (%d/%d)\n", key, last, quantity)
		<-sem
		return
	}

	fmt.Printf("Updating gift %q from %d to %d\n", key, last+1, quantity)

	db, err := parser.CreateDB(dbPath)
	if err != nil {
		fmt.Printf("Failed to open DB for %q: %v\n", key, err)
		<-sem
		return
	}
	defer db.Close()

	for i := last + 1; i <= quantity; i++ {
		url := fmt.Sprintf("https://t.me/nft/%s-%d", keySlug, i)
		doc, err := parser.FetchHTML(url, 3, 2*time.Second)
		if err != nil {
			fmt.Printf("Failed to fetch %s: %v\n", url, err)
			continue
		}

		info := parser.ParseGiftInfo(doc)
		err = parser.InsertGift(db, key, info["Model"], info["Backdrop"], info["Symbol"], i)
		if err != nil {
			fmt.Printf("Insert failed for %q #%d: %v\n", key, i, err)
		}

		if i%100 == 0 {
			fmt.Printf("Updated %q up to #%d\n", key, i)
		}
	}

	fmt.Printf("Finished updating %q\n", key)
	<-sem
}

func main() {
	keys, err := parser.LoadGiftsJSON("data/gifts.json")
	if err != nil {
		fmt.Printf("Failed to load gifts.json: %v\n", err)
		return
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, workerCount)

	for _, key := range keys {
		wg.Add(1)
		sem <- struct{}{}
		go updateGift(key, &wg, sem)
	}

	wg.Wait()
	fmt.Println("All gifts updated.")
}
