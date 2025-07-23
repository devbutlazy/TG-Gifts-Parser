package tui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/spinner"
)

type Model struct {
	data      map[string][]string
	keys      []string
	values    []string
	backdrops []string

	cursor     int
	viewOffset int
	state      state
	page       int
	totalPages int

	SelectedKey      string
	SelectedValue    string
	SelectedBackdrop string

	width  int
	height int

	spinner spinner.Model
	results []int
	error   error
}

func InitialModel() Model {
	data, keys, err := LoadData("data/gifts.json")
	if err != nil {
		fmt.Println("Error loading gifts.json:", err)
		os.Exit(1)
	}

	backdrops := LoadBackdrops("data/base.json")

	return Model{
		data:      data,
		keys:      keys,
		backdrops: backdrops,
		state:     mainMenu,
	}
}
