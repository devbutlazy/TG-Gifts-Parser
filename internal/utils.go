package internal

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

func UpdateAll() {
	fmt.Println("Stashing changes (if any)...")
	exec.Command("git", "stash", "--include-untracked").Run()

	fmt.Println("Pulling latest changes with rebase...")
	cmd := exec.Command("git", "pull", "--rebase")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("git pull --rebase failed:", err)
		return
	}

	fmt.Println("Applying stashed changes...")
	exec.Command("git", "stash", "pop").Run()
}

func ClearScreen() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Println("Failed to clear screen:", err)
	}
}
