package internal

import (
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
)

type GitHubFile struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Path string `json:"path"`
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
	s.Suffix = " Updating gift databases..."
	s.Start()

	for i, file := range dbFiles {
		s.Suffix = fmt.Sprintf(" [%d/%d] %s", i+1, len(dbFiles), file.Name)

		remoteURL := rawPrefix + file.Path
		localPath := filepath.Join(localDir, file.Name)

		fileResp, err := http.Get(remoteURL)
		if err != nil {
			s.Stop()
			fmt.Printf("❌ Failed to download %s: %v\n", file.Name, err)
			s.Start()
			continue
		}
		defer fileResp.Body.Close()

		if fileResp.StatusCode != 200 {
			s.Stop()
			fmt.Printf("⚠️  Skipping %s (status %d)\n", file.Name, fileResp.StatusCode)
			s.Start()
			continue
		}

		out, err := os.Create(localPath)
		if err != nil {
			s.Stop()
			fmt.Printf("❌ Failed to create file %s: %v\n", localPath, err)
			s.Start()
			continue
		}

		_, err = io.Copy(out, fileResp.Body)
		out.Close()
		if err != nil {
			s.Stop()
			fmt.Printf("❌ Failed to save %s: %v\n", file.Name, err)
			s.Start()
			continue
		}

		s.Stop()
		fmt.Printf("✅ [%d/%d] %s updated\n", i+1, len(dbFiles), file.Name)
		s.Start()
	}

	s.Stop()

	fmt.Println("🎉 All database files updated successfully. Starting the program in 10 seconds.")
	time.Sleep(10 * time.Second)
	ClearScreen()

	return nil
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
