package main

import (
	"fmt"
	"os"

	"tg-gifts-parser/external"
	"tg-gifts-parser/internal/misc"
	"tg-gifts-parser/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	misc.ClearScreen()

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--update":
			misc.UpdateDB()
		case "--external":
			external.ScheduleUpdater()
			return
		}
	}

	prog := tea.NewProgram(tui.InitialModel())
	if _, err := prog.Run(); err != nil {
		fmt.Println("TUI exited with error:", err)
	}
}
