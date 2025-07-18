package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type state int

const (
	mainMenu state = iota
	selectingGift
	selectingModel
	selectingBackdrop

	viewSize = 10
)

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			m.moveCursorUp()

		case "down", "j":
			m.moveCursorDown()

		case "enter":
			return m.handleEnter()

		case "backspace":
			m.handleBackspace()
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
		length = len(m.keys)
	case selectingModel:
		length = len(m.values)
	case selectingBackdrop:
		length = len(m.backdrops)
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
		if m.cursor >= m.viewOffset+viewSize {
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
		case 1:
			m.state = selectingBackdrop
		case 3:
			if m.SelectedKey != "" && m.SelectedValue != "" && m.SelectedBackdrop != "" {
				return m, func() tea.Msg {
					fmt.Printf("\n\n🚀 Starting with: %s → %s + %s\n\n", m.SelectedKey, m.SelectedValue, m.SelectedBackdrop)
					return tea.Quit()
				}
			}
		}
	case selectingGift:
		m.SelectedKey = m.keys[m.cursor]
		m.values = m.data[m.SelectedKey]
		m.state = selectingModel
	case selectingModel:
		m.SelectedValue = m.values[m.cursor]
		m.state = mainMenu
	case selectingBackdrop:
		m.SelectedBackdrop = m.backdrops[m.cursor]
		m.state = mainMenu
	}

	m.cursor, m.viewOffset = 0, 0
	return m, nil
}

func (m *Model) handleBackspace() {
	switch m.state {
	case selectingModel:
		m.state = selectingGift
	case selectingBackdrop, selectingGift:
		m.state = mainMenu
	}
	m.cursor, m.viewOffset = 0, 0
}
