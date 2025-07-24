package tui

import (
	"fmt"
	"strings"

	"tg-gifts-parser/internal/tui/utils"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	var content string
	var showFooter bool = true

	switch m.state {
	case mainMenu:
		content = m.viewMainMenu()
	case selectingGift, selectingModel:
		content = m.viewGiftSelection()
	case selectingBackdrop:
		content = m.viewBackdropSelection()
	case selectingSymbols:
		content = m.viewSymbolsSelection()
	case loadingResults:
		content = m.viewLoading()
		showFooter = false
	case viewingResults:
		content = m.viewResults()
	default:
		content = "Unknown state"
	}

	centered := centerContent(m, content)
	if showFooter {
		footerText := "Press q to quit. Use ↑/↓ and Enter to navigate. Ctrl+F to search."
		if m.state == viewingResults {
			footerText = fmt.Sprintf("Page %d/%d: Use ←/→ and ↑/↓", m.page+1, m.totalPages)
		}
		footer := footerView(m, footerText)
		return centered + "\n\n" + footer
	}
	return centered
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

func renderSelectionList(cursor, viewOffset int, items []string, header string, searchActive bool, searchQuery string) string {
	var content string
	if searchActive {
		searchStyle := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("205")).
			Padding(0, 1).
			MarginBottom(1)
		searchLine := searchStyle.Render(fmt.Sprintf("Search: %s█", searchQuery))
		content = searchLine + "\n\n" + header
	} else {
		content = header
	}

	if len(items) == 0 {
		content += "\n\n" + errorStyle.Render("No results")
	} else {
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

		content += "\n\n" + strings.Join(lines, "\n")
	}

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
		case 2:
			if m.SelectedSymbol != "" {
				line += selectedStyle.Render(fmt.Sprintf("  ✅ %s", m.SelectedSymbol))
			}
		case 3:
			if !(m.SelectedKey != "" && m.SelectedValue != "" && (m.SelectedBackdrop != "" || m.SelectedSymbol != "")) {
				line = disabledStyle.Render(line)
			}
		}

		items = append(items, line)
	}

	header := headerStyle.Render("🧭 Main Menu (↑/↓ + Enter):")
	content := header + "\n\n" + strings.Join(items, "\n")
	return boxStyle.Render(content)
}

func (m Model) viewSymbolsSelection() string {
	header := headerStyle.Render("🔣 Select a Symbol (↑/↓ and Enter, ⌫ to go back, Ctrl+F to search):")
	return renderSelectionList(m.cursor, m.viewOffset, m.filteredSymbols, header, m.searchActive, m.searchQuery)
}

func (m Model) viewGiftSelection() string {
	if m.state == selectingGift {
		header := headerStyle.Render("🎁 Select a Gift (↑/↓ and Enter, Ctrl+F to search):")
		return renderSelectionList(m.cursor, m.viewOffset, m.filteredKeys, header, m.searchActive, m.searchQuery)
	}

	header := fmt.Sprintf(
		"%s\n%s",
		headerStyle.Render(fmt.Sprintf("🎁 %s", m.SelectedKey)),
		lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Render("📦 Select a Model (↑/↓ and Enter, ⌫ to go back, Ctrl+F to search):"),
	)

	return renderSelectionList(m.cursor, m.viewOffset, m.filteredValues, header, m.searchActive, m.searchQuery)
}

func (m Model) viewBackdropSelection() string {
	header := headerStyle.Render("🖼️ Select a Backdrop (↑/↓ and Enter, ⌫ to go back, Ctrl+F to search):")
	return renderSelectionList(m.cursor, m.viewOffset, m.filteredBackdrops, header, m.searchActive, m.searchQuery)
}

func (m Model) viewLoading() string {
	var comboParts []string
	comboParts = append(comboParts, m.SelectedValue)

	if m.SelectedBackdrop != "" {
		comboParts = append(comboParts, m.SelectedBackdrop)
	}

	if m.SelectedSymbol != "" {
		comboParts = append(comboParts, m.SelectedSymbol)
	}

	comboStr := strings.Join(comboParts, " + ")

	header := headerStyle.Render(fmt.Sprintf("🔍 Searching for: %s → %s", m.SelectedKey, comboStr))
	content := fmt.Sprintf("%s\n\n%s Loading...", header, m.spinner.View())
	newBoxStyle := boxStyle
	newBoxStyle = newBoxStyle.BorderForeground(lipgloss.Color("205"))
	return newBoxStyle.Render(content)
}

func (m Model) viewResults() string {
	var comboParts []string
	comboParts = append(comboParts, m.SelectedValue)

	if m.SelectedBackdrop != "" {
		comboParts = append(comboParts, m.SelectedBackdrop)
	}

	if m.SelectedSymbol != "" {
		comboParts = append(comboParts, m.SelectedSymbol)
	}

	comboStr := strings.Join(comboParts, " + ")

	header := headerStyle.Render(fmt.Sprintf(
		"🎉 Results for: %s → %s (Page %d/%d)",
		m.SelectedKey, comboStr, m.page+1, m.totalPages,
	))

	var content string

	if m.error != nil {
		content = errorStyle.Render(fmt.Sprintf("Error: %v", m.error))
	} else if len(m.results) == 0 {
		content = errorStyle.Render(fmt.Sprintf("No matches found for: %s", comboStr))
	} else {
		start := m.page * viewSize
		end := start + viewSize
		if end > len(m.results) {
			end = len(m.results)
		}

		var links []string
		for i, entry := range m.results[start:end] {
			url := fmt.Sprintf("https://t.me/nft/%s-%d", utils.SanitizeGiftName(m.SelectedKey), entry)
			clickableLink := fmt.Sprintf("%d. %s", start+i+1, url)
			links = append(links, clickableLink)
		}

		content = header + "\n\n" + strings.Join(links, "\n")
	}

	options := []string{"Try Again", "Exit"}
	var optionLines []string
	for i, opt := range options {
		if m.cursor == i {
			prefix := cursorStyle.Render(">")
			optionLines = append(optionLines, highlightStyle.Render(fmt.Sprintf("%s %s", prefix, opt)))
		} else {
			optionLines = append(optionLines, fmt.Sprintf("  %s", selectedStyle.Render(opt)))
		}
	}

	body := fmt.Sprintf("%s\n\n%s", content, strings.Join(optionLines, "\n"))
	newBoxStyle := boxStyle
	newBoxStyle = newBoxStyle.BorderForeground(lipgloss.Color("33")).Padding(1, 2)
	return newBoxStyle.Render(body)
}
