package external

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"tg-gifts-parser/internal/parser"

	"github.com/xitongsys/parquet-go-source/local"
	"github.com/xitongsys/parquet-go/reader"
	"github.com/xitongsys/parquet-go/writer"
)

const (
	dbFolder        = "data/database"
	giftsJSONPath   = "data/gifts.json"
	updateThreshold = 10000
)

type Gift struct {
	ID       int32  `parquet:"name=id, type=INT32"`
	Name     string `parquet:"name=name, type=BYTE_ARRAY, convertedtype=UTF8"`
	Number   int32  `parquet:"name=number, type=INT32"`
	Model    string `parquet:"name=model, type=BYTE_ARRAY, convertedtype=UTF8"`
	Backdrop string `parquet:"name=backdrop, type=BYTE_ARRAY, convertedtype=UTF8"`
	Symbol   string `parquet:"name=symbol, type=BYTE_ARRAY, convertedtype=UTF8"`
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

func getExistingCount(path string) (int, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return 0, nil
	}
	fr, err := local.NewLocalFileReader(path)
	if err != nil {
		return 0, err
	}
	defer fr.Close()

	pr, err := reader.NewParquetReader(fr, new(Gift), 1)
	if err != nil {
		return 0, err
	}
	defer pr.ReadStop()

	return int(pr.GetNumRows()), nil
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
	gifts := make([]Gift, num)
	if num > 0 {
		if err := pr.Read(&gifts); err != nil {
			return nil, err
		}
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

func updateGiftIfNeeded(key string) (int, error) {
	keySlug := parser.SanitizeKey(key)
	parquetPath := filepath.Join(dbFolder, keySlug+".parquet")

	if err := ensureParquetFile(parquetPath); err != nil {
		return 0, fmt.Errorf("ensure parquet file for %q: %w", key, err)
	}

	existingCount, err := getExistingCount(parquetPath)
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

	allGifts, err := readAllGifts(parquetPath)
	if err != nil {
		return 0, fmt.Errorf("read gifts: %w", err)
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
		newGift := Gift{
			ID:       int32(i),
			Name:     key,
			Number:   int32(i),
			Model:    info["Model"],
			Backdrop: info["Backdrop"],
			Symbol:   info["Symbol"],
		}
		allGifts = append(allGifts, newGift)
		newItemsCount++

		if newItemsCount%1000 == 0 {
			fmt.Printf("Parsed %q gift item #%d\n", key, i)
		}
	}

	if err := writeAllGifts(parquetPath, allGifts); err != nil {
		return 0, fmt.Errorf("write parquet: %w", err)
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

	commitMsg := fmt.Sprintf("chore(data/database/*.parquet): updated %d gifts", newItems)
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
