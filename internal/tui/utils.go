package tui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	json "github.com/goccy/go-json"
	"github.com/iancoleman/orderedmap"
)

func LoadData(path string) (map[string][]string, []string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	omap := orderedmap.New()
	if err := json.Unmarshal(raw, omap); err != nil {
		return nil, nil, err
	}

	data := make(map[string][]string)
	keys := omap.Keys()

	for _, k := range keys {
		v, _ := omap.Get(k)

		var items []string
		b, _ := json.Marshal(v)
		_ = json.Unmarshal(b, &items)

		data[k] = items
	}

	return data, keys, nil
}

func LoadBackdrops(path string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("Error loading base.json:", err)
		os.Exit(1)
	}
	var parsed struct {
		Backdrops []string `json:"backdrops"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		fmt.Println("Failed to parse base.json:", err)
		os.Exit(1)
	}
	return parsed.Backdrops
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
