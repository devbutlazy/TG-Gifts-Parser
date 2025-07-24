package tui

import (
	"fmt"
	"os"

	"tg-gifts-parser/internal/tui/utils"

	"github.com/charmbracelet/bubbles/spinner"
)

type Model struct {
	data      map[string][]string
	keys      []string
	values    []string
	backdrops []string
	symbols   []string

	cursor     int
	viewOffset int
	state      state
	page       int
	totalPages int

	SelectedKey      string
	SelectedValue    string
	SelectedBackdrop string
	SelectedSymbol   string

	width  int
	height int

	spinner spinner.Model
	results []int
	error   error

	searchActive      bool
	searchQuery       string
	filteredKeys      []string
	filteredValues    []string
	filteredBackdrops []string
	filteredSymbols   []string
}

func InitialModel() Model {
	data, keys, err := utils.LoadData("data/gifts.json")
	if err != nil {
		fmt.Println("Error loading gifts.json:", err)
		os.Exit(1)
	}

	backdrops, symbols := utils.LoadBaseData("data/base.json")

	return Model{
		data:              data,
		keys:              keys,
		backdrops:         backdrops,
		symbols:           symbols,
		state:             mainMenu,
		filteredKeys:      keys,
		filteredValues:    []string{},
		filteredBackdrops: backdrops,
		filteredSymbols:   symbols,
	}
}
