package internal

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

func UpdateAll() {
	fmt.Println("Stashing changes (if any)...")
	stashCmd := exec.Command("git", "stash", "--include-untracked")
	if err := stashCmd.Run(); err != nil {
		fmt.Println("Failed to stash changes:", err)
		return
	}

	fmt.Println("Pulling latest changes with rebase...")
	pullCmd := exec.Command("git", "pull", "--rebase")
	pullCmd.Stdout = os.Stdout
	pullCmd.Stderr = os.Stderr
	if err := pullCmd.Run(); err != nil {
		fmt.Println("git pull --rebase failed:", err)
		fmt.Println("You can resolve the conflict manually and then run 'git rebase --continue'")
		return
	}

	fmt.Println("Applying stashed changes...")
	popCmd := exec.Command("git", "stash", "pop")
	popCmd.Stdout = os.Stdout
	popCmd.Stderr = os.Stderr
	if err := popCmd.Run(); err != nil {
		fmt.Println("git stash pop failed:", err)
		fmt.Println("There may be merge conflicts. Please resolve them manually.")
		return
	}

	fmt.Println("Update completed successfully.")
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
