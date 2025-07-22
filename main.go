package main

import (
	"fmt"

	"tg-gifts-parser/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	tui.ClearScreen()

	prog := tea.NewProgram(tui.InitialModel())
	if _, err := prog.Run(); err != nil {
		fmt.Println("TUI exited with error:", err)
	}
}

// package main

// import (
// 	"log"

// 	"tg-gifts-parser/internal/parser"
// )

// func main() {
// 	if err := parser.ParseAllGifts(); err != nil {
// 		log.Fatalf("Error: %v", err)
// 	}
// }
