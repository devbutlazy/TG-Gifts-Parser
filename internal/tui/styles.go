package tui

import "github.com/charmbracelet/lipgloss"

var (
	cursorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	selectedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Bold(true)
	headerStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Bold(true).Underline(true)
	boxStyle       = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2).BorderForeground(lipgloss.Color("240"))
	highlightStyle = lipgloss.NewStyle().Background(lipgloss.Color("57")).Foreground(lipgloss.Color("230"))
	disabledStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	mainMenuItems  = []string{"🎁 Gift", "🖼️ Backdrop", "🔣 Symbols (soon)", "🚀 Start"}
)
