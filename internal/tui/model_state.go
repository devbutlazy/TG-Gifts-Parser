package tui

import (
	"database/sql"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	_ "github.com/mattn/go-sqlite3"
)

type state int

const (
	mainMenu state = iota
	selectingGift
	selectingModel
	selectingBackdrop
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

		if m.searchActive && (m.state == selectingGift || m.state == selectingModel || m.state == selectingBackdrop) {
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
			if m.state == selectingGift || m.state == selectingModel || m.state == selectingBackdrop {
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
		case 3:
			m.cursor = 1 // skip <<symbols>>
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
	case viewingResults:
		length = 2 // Only Try Again, Exit
	}

	if m.state == mainMenu {
		switch m.cursor {
		case 0:
			m.cursor = 1
		case 1:
			if m.SelectedKey != "" && m.SelectedValue != "" && m.SelectedBackdrop != "" {
				m.cursor = 3 // skip Symbols → Start
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
		case 3:
			if m.SelectedKey != "" && m.SelectedValue != "" && m.SelectedBackdrop != "" {
				m.state = loadingResults
				m.spinner = spinner.New()
				m.spinner.Spinner = spinner.Dot
				m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
				return m, tea.Batch(
					m.spinner.Tick,
					tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
						giftDB := "database/" + SanitizeGiftName(m.SelectedKey) + ".db"
						modelName := RemovePercent(m.SelectedValue)
						backdropName := RemovePercent(m.SelectedBackdrop)
						entries, err := queryMatchingEntries(giftDB, modelName, backdropName)
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
	}

	m.cursor, m.viewOffset = 0, 0
	return m, nil
}

func (m *Model) handleBackspace() {
	switch m.state {
	case selectingModel:
		m.state = selectingGift
		m.filteredKeys = m.keys
	case selectingBackdrop, selectingGift:
		m.state = mainMenu
		m.filteredBackdrops = m.backdrops
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
	}
}

func (m *Model) resetFilteredLists() {
	m.filteredKeys = m.keys
	m.filteredValues = m.values
	m.filteredBackdrops = m.backdrops
}

func queryMatchingEntries(dbPath, model, backdrop string) ([]int, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query("SELECT number, model, backdrop FROM gifts")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []int
	for rows.Next() {
		var number int
		var dbModel, dbBackdrop string
		err := rows.Scan(&number, &dbModel, &dbBackdrop)
		if err != nil {
			return nil, err
		}

		if RemovePercent(dbModel) == model && RemovePercent(dbBackdrop) == backdrop {
			matches = append(matches, number)
		}
	}

	return matches, nil
}
