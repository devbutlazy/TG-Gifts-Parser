package parser

import (
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
	"github.com/xitongsys/parquet-go-source/local"
	"github.com/xitongsys/parquet-go/reader"
	"github.com/xitongsys/parquet-go/writer"
	"golang.org/x/net/html"
)

const workerCount = 10

type Gift struct {
	ID       int32  `parquet:"name=id, type=INT32"`
	Name     string `parquet:"name=name, type=BYTE_ARRAY, convertedtype=UTF8"`
	Model    string `parquet:"name=model, type=BYTE_ARRAY, convertedtype=UTF8"`
	Backdrop string `parquet:"name=backdrop, type=BYTE_ARRAY, convertedtype=UTF8"`
	Symbol   string `parquet:"name=symbol, type=BYTE_ARRAY, convertedtype=UTF8"`
	Number   int32  `parquet:"name=number, type=INT32"`
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

func ensureParquetFile(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fw, err := local.NewLocalFileWriter(path)
		if err != nil {
			return err
		}
		defer fw.Close()
		pw, err := writer.NewParquetWriter(fw, new(Gift), 1)
		if err != nil {
			return err
		}
		if err := pw.WriteStop(); err != nil {
			return err
		}
	}
	return nil
}

func readAllGifts(path string) ([]Gift, error) {
	fr, err := local.NewLocalFileReader(path)
	if err != nil {
		return nil, err
	}
	defer fr.Close()

	pr, err := reader.NewParquetReader(fr, new(Gift), 1)
	if err != nil {
		return nil, err
	}
	defer pr.ReadStop()

	num := int(pr.GetNumRows())
	if num == 0 {
		return []Gift{}, nil
	}
	gifts := make([]Gift, num)
	if err := pr.Read(&gifts); err != nil {
		return nil, err
	}
	return gifts, nil
}

func writeAllGifts(path string, gifts []Gift) error {
	fw, err := local.NewLocalFileWriter(path)
	if err != nil {
		return err
	}
	defer fw.Close()

	pw, err := writer.NewParquetWriter(fw, new(Gift), 1)
	if err != nil {
		return err
	}
	for _, g := range gifts {
		if err := pw.Write(g); err != nil {
			return err
		}
	}
	return pw.WriteStop()
}

func ParseAndSaveGift(key string, wg *sync.WaitGroup, sem chan struct{}) {
	defer wg.Done()

	keySlug := SanitizeKey(key)
	filePath := filepath.Join("data/database", keySlug+".parquet")

	fmt.Printf("Starting parsing gift %q, file: %s\n", key, filePath)

	if err := ensureParquetFile(filePath); err != nil {
		fmt.Printf("Failed to ensure parquet for %q: %v\n", key, err)
		<-sem
		return
	}

	url := fmt.Sprintf("https://t.me/nft/%s-1", keySlug)
	doc, err := FetchHTML(url, 3, 2*time.Second)
	if err != nil {
		fmt.Printf("Failed to fetch first page for %q: %v\n", key, err)
		<-sem
		return
	}

	quantityStr := ExtractGiftField(doc, "Quantity")
	if quantityStr == "Unknown" {
		quantityStr = ExtractQuantityFallback(doc)
	}
	quantity := CleanQuantity(quantityStr)
	if quantity == 0 {
		fmt.Printf("Warning: zero quantity for %q\n", key)
		<-sem
		return
	}

	fmt.Printf("Gift %q has quantity %d\n", key, quantity)

	allGifts, err := readAllGifts(filePath)
	if err != nil {
		fmt.Printf("Error reading existing gifts for %q: %v\n", key, err)
		<-sem
		return
	}
	existingCount := len(allGifts)

	for i := existingCount + 1; i <= quantity; i++ {
		giftURL := fmt.Sprintf("https://t.me/nft/%s-%d", keySlug, i)
		doc, err := FetchHTML(giftURL, 3, 2*time.Second)
		if err != nil {
			fmt.Printf("Warning: failed to fetch %s: %v\n", giftURL, err)
			continue
		}

		info := ParseGiftInfo(doc)

		newGift := Gift{
			ID:       int32(i),
			Name:     key,
			Model:    info["Model"],
			Backdrop: info["Backdrop"],
			Symbol:   info["Symbol"],
			Number:   int32(i),
		}
		allGifts = append(allGifts, newGift)

		if i%1000 == 0 {
			fmt.Printf("Parsed %q gift item #%d\n", key, i)
		}
	}

	if err := writeAllGifts(filePath, allGifts); err != nil {
		fmt.Printf("Error writing parquet for %q: %v\n", key, err)
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
	fmt.Println("All gifts parsed and saved to separate .parquet files in the 'data/database/' folder")
	return nil
}
