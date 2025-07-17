package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	selectingKey state = iota
	selectingValue
	viewSize = 10
)

var (
	cursorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	selectedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Bold(true)
	headerStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Bold(true).Underline(true)
	boxStyle       = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2).BorderForeground(lipgloss.Color("240"))
	highlightStyle = lipgloss.NewStyle().Background(lipgloss.Color("57")).Foreground(lipgloss.Color("230"))
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
	data, keys, err := LoadData("data/gifts.json")
	if err != nil {
		fmt.Println("Error loading gifts.json:", err)
		os.Exit(1)
	}

	return Model{
		data:  data,
		keys:  keys, // now ordered!
		state: selectingKey,
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
	var items []string
	var header string

	start := m.viewOffset
	end := start + viewSize

	if m.state == selectingKey {
		items = m.keys
		if end > len(items) {
			end = len(items)
		}
		header = headerStyle.Render("🎁 Select a Gift (↑/↓ and Enter):")
	} else {
		items = m.values
		if end > len(items) {
			end = len(items)
		}
		header = fmt.Sprintf(
			"%s\n%s",
			headerStyle.Render(fmt.Sprintf("🎁 %s", m.SelectedKey)),
			lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Render("📦 Select a Model (↑/↓ and Enter, ⌫ to go back):"),
		)
	}

	// Build list of items
	var itemLines []string
	for i := start; i < end; i++ {
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle.Render("→")
			itemLines = append(itemLines, highlightStyle.Render(fmt.Sprintf("%s %s", cursor, items[i])))
		} else {
			itemLines = append(itemLines, fmt.Sprintf("  %s", selectedStyle.Render(items[i])))
		}
	}

	// Combine header and items into a single box
	content := header + "\n\n" + strings.Join(itemLines, "\n")
	box := boxStyle.Render(content)

	// Final render
	var b strings.Builder
	b.WriteString(box)

	// Selected value
	if m.SelectedValue != "" {
		b.WriteString("\n\n")
		b.WriteString(selectedStyle.Render(fmt.Sprintf("✅ You selected: %s → %s", m.SelectedKey, m.SelectedValue)))
	}

	b.WriteString("\n\nPress q to quit.\n")
	return b.String()
}
