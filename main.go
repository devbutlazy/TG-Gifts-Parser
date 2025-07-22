package main

import (
	"fmt"
	"os"

	"tg-gifts-parser/internal"
	"tg-gifts-parser/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	tui.ClearScreen()

	if len(os.Args) > 1 && os.Args[1] == "--update" {
		internal.UpdateAllDatabasesFromGitHub()
	}


	prog := tea.NewProgram(tui.InitialModel())
	if _, err := prog.Run(); err != nil {
		fmt.Println("TUI exited with error:", err)
	}
}
