package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	var content string
	switch m.state {
	case mainMenu:
		content = m.viewMainMenu()
	case selectingGift, selectingModel:
		content = m.viewGiftSelection()
	case selectingBackdrop:
		content = m.viewBackdropSelection()
	default:
		content = "Unknown state"
	}

	centered := centerContent(m, content)
	footer := footerView(m, "Press q to quit. Use ↑/↓ and Enter to navigate.")

	return centered + "\n\n" + footer
}

func centerContent(m Model, content string) string {
	lines := strings.Split(content, "\n")
	contentHeight := len(lines)
	maxWidth := 0
	for _, line := range lines {
		if w := lipgloss.Width(line); w > maxWidth {
			maxWidth = w
		}
	}

	verticalPadding := 0
	horizontalPadding := 0
	if m.height > contentHeight+3 {
		verticalPadding = (m.height - contentHeight - 3) / 2
	}
	if m.width > maxWidth {
		horizontalPadding = (m.width - maxWidth) / 2
	}

	var padded []string
	for i := 0; i < verticalPadding; i++ {
		padded = append(padded, "")
	}
	for _, line := range lines {
		padded = append(padded, strings.Repeat(" ", horizontalPadding)+line)
	}
	return strings.Join(padded, "\n")
}

func footerView(m Model, text string) string {
	maxWidth := 50
	if len(text) > maxWidth {
		text = text[:maxWidth]
	}

	padding := 0
	if m.width > len(text) {
		padding = (m.width - len(text)) / 2
	}

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("238")).
		Italic(true)

	return footerStyle.Render(fmt.Sprintf("%s%s", strings.Repeat(" ", padding), text))
}

func renderSelectionList(cursor, viewOffset int, items []string, header string) string {
	start := viewOffset
	end := start + viewSize
	if end > len(items) {
		end = len(items)
	}

	var lines []string
	for i := start; i < end; i++ {
		prefix := "  "
		line := items[i]
		if i == cursor {
			prefix = cursorStyle.Render(">")
			line = highlightStyle.Render(fmt.Sprintf("%s %s", prefix, line))
		} else {
			line = fmt.Sprintf("  %s", selectedStyle.Render(line))
		}
		lines = append(lines, line)
	}

	content := header + "\n\n" + strings.Join(lines, "\n")
	return boxStyle.Render(content)
}

func (m Model) viewMainMenu() string {
	var items []string
	for i, item := range mainMenuItems {
		var line string
		cursor := "  "

		switch i {
		case m.cursor:
			cursor = cursorStyle.Render(">")
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
	return boxStyle.Render(content)
}

func (m Model) viewGiftSelection() string {
	if m.state == selectingGift {
		header := headerStyle.Render("🎁 Select a Gift (↑/↓ and Enter):")
		return renderSelectionList(m.cursor, m.viewOffset, m.keys, header)
	}

	header := fmt.Sprintf(
		"%s\n%s",
		headerStyle.Render(fmt.Sprintf("🎁 %s", m.SelectedKey)),
		lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Render("📦 Select a Model (↑/↓ and Enter, ⌫ to go back):"),
	)

	return renderSelectionList(m.cursor, m.viewOffset, m.values, header)
}

func (m Model) viewBackdropSelection() string {
	header := headerStyle.Render("🖼️ Select a Backdrop (↑/↓ and Enter, ⌫ to go back):")
	return renderSelectionList(m.cursor, m.viewOffset, m.backdrops, header)
}

