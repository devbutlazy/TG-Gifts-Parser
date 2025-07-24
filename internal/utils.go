package internal

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/briandowns/spinner"
	_ "github.com/mattn/go-sqlite3"
)

type GitHubFile struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Path string `json:"path"`
}

func countEntriesInDB(path string) (int, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM gifts").Scan(&count)
	return count, err
}

func UpdateAllDatabasesFromGitHub() error {
	const (
		apiURL    = "https://api.github.com/repos/devbutlazy/TG-Gifts-Parser/contents/data/database"
		rawPrefix = "https://raw.githubusercontent.com/devbutlazy/TG-Gifts-Parser/main/"
		localDir  = "data/database"
	)

	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf("failed to fetch file list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var files []GitHubFile
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return fmt.Errorf("failed to decode file list: %w", err)
	}

	var dbFiles []GitHubFile
	for _, file := range files {
		if file.Type == "file" && filepath.Ext(file.Name) == ".db" {
			dbFiles = append(dbFiles, file)
		}
	}

	if len(dbFiles) == 0 {
		return fmt.Errorf("no .db files found in GitHub repo")
	}

	if err := os.MkdirAll(localDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create local database directory: %w", err)
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Checking gift databases..."
	s.Start()
	defer s.Stop()

	sem := make(chan struct{}, 4)
	results := make(chan string, len(dbFiles))

	for i, file := range dbFiles {
		sem <- struct{}{}
		go func(i int, file GitHubFile) {
			defer func() { <-sem }()

			s.Suffix = fmt.Sprintf(" [%d/%d] %s", i+1, len(dbFiles), file.Name)
			localPath := filepath.Join(localDir, file.Name)

			localCount, err := countEntriesInDB(localPath)
			if err != nil {
				localCount = 0
			}

			tmpFile, err := os.CreateTemp("", "tmpdb-*.db")
			if err != nil {
				results <- fmt.Sprintf("⚠️ Failed to create temp file for %s: %v", file.Name, err)
				return
			}
			tmpPath := tmpFile.Name()
			tmpFile.Close()

			err = downloadFile(rawPrefix+file.Path, tmpPath)
			if err != nil {
				os.Remove(tmpPath)
				results <- fmt.Sprintf("❌ Failed to download remote %s: %v", file.Name, err)
				return
			}

			remoteCount, err := countEntriesInDB(tmpPath)
			if err != nil {
				os.Remove(tmpPath)
				results <- fmt.Sprintf("⚠️ Could not count remote entries in %s: %v", file.Name, err)
				return
			}

			if localCount >= remoteCount {
				os.Remove(tmpPath)
				results <- fmt.Sprintf("✅ [%d/%d] %s is up to date (%d entries)", i+1, len(dbFiles), file.Name, localCount)
				return
			}

			err = os.Rename(tmpPath, localPath)
			if err != nil {
				os.Remove(tmpPath)
				results <- fmt.Sprintf("❌ Failed to update %s: %v", file.Name, err)
				return
			}

			results <- fmt.Sprintf("⬆️ [%d/%d] %s updated (%d -> %d entries)", i+1, len(dbFiles), file.Name, localCount, remoteCount)
		}(i, file)
	}

	for i := 0; i < len(dbFiles); i++ {
		msg := <-results
		s.Stop()
		fmt.Println(msg)
		s.Start()
	}

	fmt.Println("🎉 All database files processed. Starting the program in 10 seconds.")
	time.Sleep(10 * time.Second)
	ClearScreen()

	return nil
}

func downloadFile(url, localPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	out, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func ClearScreen() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Println("Failed to clear screen:", err)
	}
}
