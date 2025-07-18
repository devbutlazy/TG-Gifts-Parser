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
	mainMenu state = iota
	selectingGift
	selectingModel
	selectingBackdrop
	viewSize = 10
)

var (
	cursorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	selectedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Bold(true)
	headerStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Bold(true).Underline(true)
	boxStyle       = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2).BorderForeground(lipgloss.Color("240"))
	highlightStyle = lipgloss.NewStyle().Background(lipgloss.Color("57")).Foreground(lipgloss.Color("230"))
	disabledStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	mainMenuItems  = []string{"🎁 Gift", "🖼️ Backdrop", "🔣 Symbols (soon)", "🚀 Start"}
)

type Model struct {
	data             map[string][]string
	keys             []string
	values           []string
	backdrops        []string
	cursor           int
	viewOffset       int
	state            state
	SelectedKey      string
	SelectedValue    string
	SelectedBackdrop string
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
			if m.state == mainMenu {
				switch m.cursor {
				case 1:
					m.cursor = 0
				case 3:
					m.cursor = 1 // skip Symbols → go to Backdrop
				}

			} else {
				if m.cursor > 0 {
					m.cursor--
					if m.cursor < m.viewOffset {
						m.viewOffset--
					}
				}
			}

		case "down", "j":
			length := 0
			switch m.state {
			case mainMenu:
				length = len(mainMenuItems)
				switch m.cursor {
				case 1: // from Backdrop
					if m.SelectedKey != "" && m.SelectedValue != "" && m.SelectedBackdrop != "" {
						m.cursor = 3 // skip Symbols → go to Start
					}
				case 0: // else: stay on 1
					m.cursor = 1
				}
			case selectingGift:
				length = len(m.keys)
			case selectingModel:
				length = len(m.values)
			case selectingBackdrop:
				length = len(m.backdrops)
			}

			if m.state != mainMenu && m.cursor < length-1 {
				m.cursor++
				if m.cursor >= m.viewOffset+viewSize {
					m.viewOffset++
				}
			}

		case "enter":
			switch m.state {
			case mainMenu:
				switch m.cursor {
				case 0:
					m.state = selectingGift
					m.cursor, m.viewOffset = 0, 0
				case 1:
					m.state = selectingBackdrop
					m.cursor, m.viewOffset = 0, 0
				case 3:
					if m.SelectedKey != "" && m.SelectedValue != "" && m.SelectedBackdrop != "" {
						return m, func() tea.Msg {
							fmt.Printf("\n🚀 Starting with: %s → %s + %s\n\n", m.SelectedKey, m.SelectedValue, m.SelectedBackdrop)
							return tea.Quit()
						}
					}
				}
			case selectingGift:
				m.SelectedKey = m.keys[m.cursor]
				m.values = m.data[m.SelectedKey]
				m.cursor, m.viewOffset = 0, 0
				m.state = selectingModel

			case selectingModel:
				m.SelectedValue = m.values[m.cursor]
				m.state = mainMenu
				m.cursor, m.viewOffset = 0, 0

			case selectingBackdrop:
				m.SelectedBackdrop = m.backdrops[m.cursor]
				m.state = mainMenu
				m.cursor, m.viewOffset = 0, 0
			}

		case "backspace":
			switch m.state {
			case selectingModel:
				m.state = selectingGift
				m.cursor, m.viewOffset = 0, 0
			case selectingBackdrop, selectingGift:
				m.state = mainMenu
				m.cursor, m.viewOffset = 0, 0
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	switch m.state {
	case mainMenu:
		return m.viewMainMenu()
	case selectingGift, selectingModel:
		return m.viewGiftSelection()
	case selectingBackdrop:
		return m.viewBackdropSelection()
	}
	return "Unknown state"
}

func (m Model) viewMainMenu() string {
	var items []string
	for i, item := range mainMenuItems {
		cursor := "  "
		line := item

		switch i {
		case m.cursor:
			cursor = cursorStyle.Render("→")
			line = highlightStyle.Render(fmt.Sprintf("%s %s", cursor, item))
		case 2:
			line = disabledStyle.Render(fmt.Sprintf("  %s", item)) // grayed out
		default:
			line = fmt.Sprintf("  %s", item)
		}

		switch i {
		case 0:
			if m.SelectedKey != "" && m.SelectedValue != "" {
				line += selectedStyle.Render(fmt.Sprintf("  ✅ %s → %s", m.SelectedKey, m.SelectedValue))
			}
		case 1:
			if m.SelectedBackdrop != "" {
				line += selectedStyle.Render(fmt.Sprintf("  ✅ %s", m.SelectedBackdrop))
			}
		case 3:
			if !(m.SelectedKey != "" && m.SelectedValue != "" && m.SelectedBackdrop != "") {
				line = disabledStyle.Render(line)
			}
		}

		items = append(items, line)
	}

	header := headerStyle.Render("🧭 Main Menu (↑/↓ + Enter):")
	content := header + "\n\n" + strings.Join(items, "\n")
	return boxStyle.Render(content) + "\n\nPress q to quit.\n"
}

func (m Model) viewGiftSelection() string {
	var items []string
	var header string
	start := m.viewOffset
	end := start + viewSize

	if m.state == selectingGift {
		items = m.keys
		header = headerStyle.Render("🎁 Select a Gift (↑/↓ and Enter):")
	} else {
		items = m.values
		header = fmt.Sprintf(
			"%s\n%s",
			headerStyle.Render(fmt.Sprintf("🎁 %s", m.SelectedKey)),
			lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Render("📦 Select a Model (↑/↓ and Enter, ⌫ to go back):"),
		)
	}
	if end > len(items) {
		end = len(items)
	}

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

	content := header + "\n\n" + strings.Join(itemLines, "\n")
	box := boxStyle.Render(content)
	return box + "\n\nPress q to quit.\n"
}

func (m Model) viewBackdropSelection() string {
	start := m.viewOffset
	end := start + viewSize
	if end > len(m.backdrops) {
		end = len(m.backdrops)
	}
	header := headerStyle.Render("🖼️ Select a Backdrop (↑/↓ and Enter, ⌫ to go back):")

	var itemLines []string
	for i := start; i < end; i++ {
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle.Render("→")
			itemLines = append(itemLines, highlightStyle.Render(fmt.Sprintf("%s %s", cursor, m.backdrops[i])))
		} else {
			itemLines = append(itemLines, fmt.Sprintf("  %s", selectedStyle.Render(m.backdrops[i])))
		}
	}

	content := header + "\n\n" + strings.Join(itemLines, "\n")
	box := boxStyle.Render(content)
	return box + "\n\nPress q to quit.\n"
}
