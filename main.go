package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"goblin-terminal/internal/game"
	"goblin-terminal/internal/ui"
	"goblin-terminal/pkg/docker"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Flags
	// Flags
	questFlag := flag.Int("quest", 0, "Jump to specific quest ID (debug)")
	resetFlag := flag.Bool("reset", false, "Reset save data")
	hardFlag := flag.Bool("hard", false, "Enable Hard Mode (no command hints)")
	flag.Parse()

	// 1. Initialize Container Manager
	// We use a fixed name for the game container
	manager, err := docker.NewManager("goblin-terminal:latest", "goblin-game")
	if err != nil {
		fmt.Printf("Error initializing container manager: %v\n", err)
		os.Exit(1)
	}

	// Handle Reset
	if *resetFlag {
		if err := game.ResetState(); err != nil {
			fmt.Printf("Error resetting state: %v\n", err)
		} else {
			fmt.Println("Save state reset.")
		}

		if err := manager.ResetStorage(); err != nil {
			fmt.Printf("Error resetting storage: %v\n", err)
		} else {
			fmt.Println("Game storage reset.")
		}
		return // Exit after reset
	}

	// 2. Load Quests
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	questsPath := filepath.Join(cwd, "quests", "quests.yaml")
	quests, err := game.LoadQuests(questsPath)
	if err != nil {
		fmt.Printf("Error loading quests: %v\n", err)
		os.Exit(1)
	}

	// Determine starting quest index
	startQuestIdx := 0

	// properties of LoadState
	state, err := game.LoadState()
	if err == nil {
		startQuestIdx = state.CurrentQuestID
	}

	// Flag overrides save
	if *questFlag > 0 {
		// Assuming 1-based IDs map to 0-based index
		// For robustness, usually we'd search ID, but math is safe for this prototype
		startQuestIdx = *questFlag - 1
	}

	// 3. Start TUI
	// The construction of the Image and Container will happen inside the UI for better feedback
	p := tea.NewProgram(ui.NewModel(quests, manager, startQuestIdx, *hardFlag), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
