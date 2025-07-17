package main

import (
	"fmt"
	// "log"

	// "tg-gifts-parser/internal/parser"
	"tg-gifts-parser/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// fmt.Print("~> Enter the gift url: ")
	// var url string
	// fmt.Scan(&url)

	// doc, err := parser.FetchHTML(url)
	// if err != nil {
	// 	log.Fatalf("Error: %v", err)
	// }

	// info := parser.ParseGiftInfo(doc)
	// keys := []string{"Owner", "Quantity", "Model", "Backdrop", "Symbol"}

	// for _, key := range keys {
	// 	if key == "Quantity" {
	// 		fmt.Printf("%s: %d\n", key, parser.CleanQuantity(info[key]))
	// 	} else {
	// 		fmt.Printf("%s: %s\n", key, info[key])
	// 	}
	// }

	// fmt.Println("\nNow launching TUI selector...")
	prog := tea.NewProgram(tui.InitialModel())
	if _, err := prog.Run(); err != nil {
		fmt.Println("TUI exited with error:", err)
	}
}
