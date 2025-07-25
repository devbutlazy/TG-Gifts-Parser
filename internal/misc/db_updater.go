package misc

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/briandowns/spinner"
)

func loadLocalHashes(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, err
	}
	var m map[string]string
	err = json.Unmarshal(data, &m)
	return m, err
}

func saveLocalHashes(path string, hashes map[string]string) error {
	data, err := json.MarshalIndent(hashes, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP status %d", resp.StatusCode)
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

func UpdateDB() error {
	const (
		remoteHashesURL = "https://raw.githubusercontent.com/devbutlazy/TG-Gifts-Parser/main/data/hashes.json"
		rawPrefix       = "https://raw.githubusercontent.com/devbutlazy/TG-Gifts-Parser/main/data/database/"
		localDir        = "data/database"
		localHashesPath = "data/hashes.json"
	)

	resp, err := http.Get(remoteHashesURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("remote hashes.json returned status %d", resp.StatusCode)
	}

	var remoteHashes map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&remoteHashes); err != nil {
		return err
	}

	localHashes, err := loadLocalHashes(localHashesPath)
	if err != nil {
		return err
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Comparing database files..."
	s.Start()
	defer s.Stop()

	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	var mu sync.Mutex
	results := make([]string, 0, len(remoteHashes))

	for i, fName := range sortedKeys(remoteHashes) {
		wg.Add(1)
		sem <- struct{}{}

		go func(idx int, name string) {
			defer wg.Done()
			defer func() { <-sem }()

			s.Suffix = fmt.Sprintf(" [%d/%d] %s", idx+1, len(remoteHashes), name)
			localHash := localHashes[name]
			remoteHash := remoteHashes[name]
			localPath := filepath.Join(localDir, name)

			if localHash == remoteHash {
				mu.Lock()
				results = append(results, fmt.Sprintf("✅ [%d/%d] %s is up to date", idx+1, len(remoteHashes), name))
				mu.Unlock()
				return
			}

			tmpFile, err := os.CreateTemp("", "tmpdb-*.db")
			if err != nil {
				mu.Lock()
				results = append(results, fmt.Sprintf("⚠️ failed to create temp file for %s: %v", name, err))
				mu.Unlock()
				return
			}
			tmpPath := tmpFile.Name()
			tmpFile.Close()

			if err := downloadFile(rawPrefix+name, tmpPath); err != nil {
				os.Remove(tmpPath)
				mu.Lock()
				results = append(results, fmt.Sprintf("❌ failed to download %s: %v", name, err))
				mu.Unlock()
				return
			}

			if err := os.Rename(tmpPath, localPath); err != nil {
				os.Remove(tmpPath)
				mu.Lock()
				results = append(results, fmt.Sprintf("❌ failed to update %s: %v", name, err))
				mu.Unlock()
				return
			}

			mu.Lock()
			localHashes[name] = remoteHash
			results = append(results, fmt.Sprintf("⬆️ [%d/%d] %s updated", idx+1, len(remoteHashes), name))
			mu.Unlock()
		}(i, fName)
	}

	wg.Wait()
	s.Stop()

	for _, msg := range results {
		fmt.Println(msg)
	}

	return saveLocalHashes(localHashesPath, localHashes)
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
