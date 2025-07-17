package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type state int

const (
	selectingKey state = iota
	selectingValue
	viewSize = 10
)

type Model struct {
	data          map[string][]string
	keys          []string
	values        []string
	cursor        int
	viewOffset    int
	state         state
	SelectedKey   string
	SelectedValue string
}

func InitialModel() Model {
	data, err := LoadData("data/gifts.json")
	if err != nil {
		fmt.Println("Error loading gifts.json:", err)
		os.Exit(1)
	}

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}

	return Model{
		data:   data,
		keys:   keys,
		state:  selectingKey,
		cursor: 0,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.viewOffset {
					m.viewOffset--
				}
			}
		case "down", "j":
			length := len(m.keys)
			if m.state == selectingValue {
				length = len(m.values)
			}
			if m.cursor < length-1 {
				m.cursor++
				if m.cursor >= m.viewOffset+viewSize {
					m.viewOffset++
				}
			}
		case "enter":
			switch m.state {
			case selectingKey:
				m.SelectedKey = m.keys[m.cursor]
				m.values = m.data[m.SelectedKey]
				m.cursor = 0
				m.viewOffset = 0
				m.state = selectingValue
			case selectingValue:
				m.SelectedValue = m.values[m.cursor]
				return m, tea.Quit
			}
		case "backspace":
			if m.state == selectingValue {
				m.state = selectingKey
				m.cursor = 0
				m.viewOffset = 0
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	var b strings.Builder

	start := m.viewOffset
	end := start + viewSize
	if m.state == selectingKey && end > len(m.keys) {
		end = len(m.keys)
	}
	if m.state == selectingValue && end > len(m.values) {
		end = len(m.values)
	}

	items := m.keys
	if m.state == selectingValue {
		items = m.values
	}

	header := "Select a key (use ↑/↓ and Enter):\n\n"
	if m.state == selectingValue {
		header = fmt.Sprintf("Key: %s\nSelect a value (use ↑/↓ and Enter, Backspace to go back):\n\n", m.SelectedKey)
	}
	b.WriteString(header)

	for i := start; i < end; i++ {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}
		fmt.Fprintf(&b, "%s %s\n", cursor, items[i])
	}

	if m.SelectedValue != "" {
		fmt.Fprintf(&b, "\nYou selected: %s -> %s\n", m.SelectedKey, m.SelectedValue)
	}
	b.WriteString("\nPress q to quit.\n")

	return b.String()
}
