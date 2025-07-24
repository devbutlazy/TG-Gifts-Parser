package tui

import (
	"math"
	"strings"
	"time"

	"tg-gifts-parser/internal/tui/utils"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	mainMenu state = iota
	selectingGift
	selectingModel
	selectingBackdrop
	selectingSymbols
	loadingResults
	viewingResults
	viewSize = 10
)

type loadingMsg struct{}

type resultsMsg struct {
	entries []int
	err     error
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case tea.KeyMsg:
		if m.searchActive && (m.state == selectingGift || m.state == selectingModel || m.state == selectingBackdrop || m.state == selectingSymbols) {
			switch msg.String() {
			case "ctrl+f":
				m.searchActive = false
				m.searchQuery = ""
				m.resetFilteredLists()
				return m, nil
			case "backspace":
				if m.searchQuery != "" {
					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
					m.filterLists()
					m.cursor, m.viewOffset = 0, 0
				} else {
					m.handleBackspace()
				}
				return m, nil
			case "enter":
				return m.handleEnter()
			default:
				if len(msg.String()) == 1 && msg.String()[0] >= 32 && msg.String()[0] <= 126 {
					m.searchQuery += msg.String()
					m.filterLists()
					m.cursor, m.viewOffset = 0, 0
					return m, nil
				}
			}
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "ctrl+f":
			if m.state == selectingGift || m.state == selectingModel || m.state == selectingBackdrop || m.state == selectingSymbols {
				m.searchActive = true
				return m, nil
			}

		case "up", "k":
			m.moveCursorUp()

		case "down", "j":
			m.moveCursorDown()

		case "left", "h":
			if m.state == viewingResults && m.page > 0 {
				m.page--
				m.cursor = 0
			}

		case "right", "l":
			if m.state == viewingResults && m.page < m.totalPages-1 {
				m.page++
				m.cursor = 0
			}

		case "enter":
			if m.state == viewingResults {
				switch m.cursor {
				case 0:
					m.state = mainMenu
					m.cursor, m.viewOffset, m.page = 0, 0, 0
					m.searchActive = false
					m.searchQuery = ""
					m.resetFilteredLists()
					return m, nil
				case 1:
					return m, tea.Quit
				}
			} else {
				return m.handleEnter()
			}

		case "backspace":
			if m.state != viewingResults {
				m.handleBackspace()
			}
		}

	case loadingMsg:
		if m.state == loadingResults {
			return m, m.spinner.Tick
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case resultsMsg:
		if m.state == loadingResults {
			m.results = msg.entries
			m.error = msg.err
			m.state = viewingResults
			m.cursor = 0
			m.viewOffset = 0
			m.page = 0
			m.totalPages = int(math.Ceil(float64(len(m.results)) / float64(viewSize)))
			m.searchActive = false
			m.searchQuery = ""
			m.resetFilteredLists()
		}
	}

	return m, nil
}

func (m *Model) moveCursorUp() {
	if m.state == mainMenu {
		switch m.cursor {
		case 1:
			m.cursor = 0
		case 2:
			m.cursor = 1
		case 3:
			if m.SelectedKey != "" && m.SelectedValue != "" && (m.SelectedBackdrop != "" || m.SelectedSymbol != "") {
				m.cursor = 2
			} else {
				m.cursor = 1
			}
		}
	} else if m.state == viewingResults {
		if m.cursor > 0 {
			m.cursor--
		}
	} else if m.cursor > 0 {
		m.cursor--
		if m.cursor < m.viewOffset {
			m.viewOffset--
		}
	}
}

func (m *Model) moveCursorDown() {
	var length int
	switch m.state {
	case mainMenu:
		length = len(mainMenuItems)
	case selectingGift:
		length = len(m.filteredKeys)
	case selectingModel:
		length = len(m.filteredValues)
	case selectingBackdrop:
		length = len(m.filteredBackdrops)
	case selectingSymbols:
		length = len(m.filteredSymbols)
	case viewingResults:
		length = 2 // Only Try Again, Exit
	}

	if m.state == mainMenu {
		switch m.cursor {
		case 0:
			m.cursor = 1
		case 1:
			m.cursor = 2
		case 2:
			if m.SelectedKey != "" && m.SelectedValue != "" && (m.SelectedBackdrop != "" || m.SelectedSymbol != "") {
				m.cursor = 3
			}
		}
	} else if m.cursor < length-1 {
		m.cursor++
		if m.state != viewingResults && m.cursor >= m.viewOffset+viewSize {
			m.viewOffset++
		}
	}
}

func (m *Model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.state {
	case mainMenu:
		switch m.cursor {
		case 0:
			m.state = selectingGift
			m.filteredKeys = m.keys
		case 1:
			m.state = selectingBackdrop
			m.filteredBackdrops = m.backdrops
		case 2:
			m.state = selectingSymbols
			m.filteredSymbols = m.symbols
		case 3:
			if m.SelectedKey != "" && m.SelectedValue != "" && (m.SelectedBackdrop != "" || m.SelectedSymbol != "") {
				m.state = loadingResults
				m.spinner = spinner.New()
				m.spinner.Spinner = spinner.Dot
				m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

				return m, tea.Batch(
					m.spinner.Tick,
					tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
						giftDB := "data/database/" + utils.SanitizeGiftName(m.SelectedKey) + ".db"
						modelName := utils.RemovePercent(m.SelectedValue)
						backdropName := utils.RemovePercent(m.SelectedBackdrop)
						symbolName := utils.RemovePercent(m.SelectedSymbol)

						entries, err := utils.QueryEntries(giftDB, modelName, backdropName, symbolName)
						return resultsMsg{entries, err}
					}),
				)
			}
		}
	case selectingGift:
		if len(m.filteredKeys) > 0 {
			m.SelectedKey = m.filteredKeys[m.cursor]
			m.values = m.data[m.SelectedKey]
			m.filteredValues = m.values
			m.state = selectingModel
			m.searchActive = false
			m.searchQuery = ""
		}
	case selectingModel:
		if len(m.filteredValues) > 0 {
			m.SelectedValue = m.filteredValues[m.cursor]
			m.state = mainMenu
			m.searchActive = false
			m.searchQuery = ""
		}
	case selectingBackdrop:
		if len(m.filteredBackdrops) > 0 {
			m.SelectedBackdrop = m.filteredBackdrops[m.cursor]
			m.state = mainMenu
			m.searchActive = false
			m.searchQuery = ""
		}
	case selectingSymbols:
		if len(m.filteredSymbols) > 0 {
			m.SelectedSymbol = m.filteredSymbols[m.cursor]
			m.state = mainMenu
			m.searchActive = false
			m.searchQuery = ""
		}
	}

	m.cursor, m.viewOffset = 0, 0
	return m, nil
}

func (m *Model) handleBackspace() {
	switch m.state {
	case selectingModel:
		m.state = selectingGift
		m.filteredKeys = m.keys
	case selectingBackdrop:
		m.state = mainMenu
		m.filteredBackdrops = m.backdrops
	case selectingSymbols:
		m.state = mainMenu
		m.filteredSymbols = m.symbols
	case selectingGift:
		m.state = mainMenu
	}
	m.cursor, m.viewOffset = 0, 0
	m.searchActive = false
	m.searchQuery = ""
}

func (m *Model) filterLists() {
	if m.searchQuery == "" {
		m.resetFilteredLists()
		return
	}

	query := strings.ToLower(m.searchQuery)
	switch m.state {
	case selectingGift:
		m.filteredKeys = nil
		for _, key := range m.keys {
			if strings.Contains(strings.ToLower(key), query) {
				m.filteredKeys = append(m.filteredKeys, key)
			}
		}
	case selectingModel:
		m.filteredValues = nil
		for _, value := range m.values {
			if strings.Contains(strings.ToLower(value), query) {
				m.filteredValues = append(m.filteredValues, value)
			}
		}
	case selectingBackdrop:
		m.filteredBackdrops = nil
		for _, backdrop := range m.backdrops {
			if strings.Contains(strings.ToLower(backdrop), query) {
				m.filteredBackdrops = append(m.filteredBackdrops, backdrop)
			}
		}
	case selectingSymbols:
		m.filteredSymbols = nil
		for _, symbol := range m.symbols {
			if strings.Contains(strings.ToLower(symbol), query) {
				m.filteredSymbols = append(m.filteredSymbols, symbol)
			}
		}
	}
}

func (m *Model) resetFilteredLists() {
	m.filteredKeys = m.keys
	m.filteredValues = m.values
	m.filteredBackdrops = m.backdrops
	m.filteredSymbols = m.symbols
}
