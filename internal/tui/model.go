package tui

import (
	"encoding/json"
	"os"
)

func LoadData(path string) (map[string][]string, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var result map[string][]string
	if err := json.Unmarshal(file, &result); err != nil {
		return nil, err
	}

	return result, nil
}
