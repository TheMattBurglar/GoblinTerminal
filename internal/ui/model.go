package ui

import (
	"fmt"
	"strings"
	"time"

	"goblin-terminal/internal/game"
	"goblin-terminal/pkg/docker"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Define custom messages
type containerReadyMsg struct{ err error }
type commandResultMsg struct {
	output string
	err    error
}

type Model struct {
	// dependencies
	quests  []game.Quest
	manager *docker.Manager

	// Game state
	currentQuestIdx int
	gameStarted     bool
	ready           bool
	output          []string // Output buffer for the virtual terminal
	lastOutput      string   // Last command output for validation
	input           string   // Current input
	history         []string // Command history
	historyIdx      int      // Current position in history
	glitchText      string   // What the goblin is currently saying

	// View state
	width, height int
	viewportReady bool // To avoid rendering before size is known
}

func NewModel(quests []game.Quest, manager *docker.Manager, startQuestID int) Model {
	initialText := "Initializing Goblin Terminal..."
	if len(quests) > 0 {
		initialText = "Loading content..."
	}

	// Ensure startQuestID is valid
	if startQuestID < 0 {
		startQuestID = 0
	}
	if startQuestID >= len(quests) {
		startQuestID = len(quests) - 1
	}

	return Model{
		quests:          quests,
		manager:         manager,
		output:          []string{initialText},
		glitchText:      "<'.'> ...",
		currentQuestIdx: startQuestID,
		history:         []string{},
		historyIdx:      0,
	}
}

func (m Model) Init() tea.Cmd {
	// Start by building/starting the container async
	return func() tea.Msg {
		m.output = append(m.output, "Building simulation environment... (this may take a moment)")
		if err := m.manager.BuildImage(); err != nil {
			return containerReadyMsg{err: err}
		}
		if err := m.manager.StartContainer(); err != nil {
			return containerReadyMsg{err: err}
		}
		return containerReadyMsg{err: nil}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewportReady = true
		return m, nil

	case containerReadyMsg:
		if msg.err != nil {
			m.output = append(m.output, fmt.Sprintf("Error starting environment: %v", msg.err))
			return m, tea.Quit
		}
		m.ready = true
		m.gameStarted = true
		m.output = append(m.output, "Environment ready.")

		// Restore environment state (users, permissions) if needed
		if err := m.manager.RestoreEnvironment(m.currentQuestIdx); err != nil {
			m.output = append(m.output, fmt.Sprintf("Warning: State restoration issue: %v", err))
		}

		// Display loaded game message if we are not at 0
		if m.currentQuestIdx > 0 {
			m.output = append(m.output, fmt.Sprintf("Resuming from Quest %d...", m.quests[m.currentQuestIdx].ID))
		}
		m.output = append(m.output, "")

		// Load quest intro
		if len(m.quests) > 0 {
			// Start with the current quest index (which might be loaded or flagged)
			return m, m.startQuest(m.currentQuestIdx)
		}
		return m, nil

	case commandResultMsg:
		// Display output
		if msg.err != nil {
			m.output = append(m.output, fmt.Sprintf("Error: %v", msg.err))
		} else {
			lines := strings.Split(msg.output, "\n")
			// Filter out empty last line often caused by split
			if len(lines) > 0 && lines[len(lines)-1] == "" {
				lines = lines[:len(lines)-1]
			}
			m.output = append(m.output, lines...)
			m.lastOutput = msg.output
		}

		// Check win condition
		return m, m.checkWinCondition()

	case tea.KeyMsg:
		if !m.ready {
			if msg.Type == tea.KeyCtrlC || msg.Type == tea.KeyEsc {
				return m, tea.Quit
			}
			return m, nil
		}

		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			// Cleanup on exit
			// Ideally we would do this in a defer or cleanup hook, but bubbletea doesn't have a global cleanup easily accessible here
			// For now, we rely on the container being --rm or stopped
			m.manager.StopContainer()
			return m, tea.Quit
		case tea.KeyEnter:
			cmdText := strings.TrimSpace(m.input)

			// Calculate display path for history
			displayPath := m.manager.CurrentDir
			if strings.HasPrefix(displayPath, "/home/player") {
				displayPath = strings.Replace(displayPath, "/home/player", "~", 1)
			}

			m.output = append(m.output, fmt.Sprintf("player@goblin:%s$ %s", displayPath, cmdText))
			m.input = ""

			// Add to history if not empty
			if cmdText != "" {
				m.history = append(m.history, cmdText)
				m.historyIdx = len(m.history) // Reset index to end
			}

			if cmdText == "exit" {
				m.output = append(m.output, "Shutting down simulation...")
				return m, tea.Sequence(
					tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
						return tea.Quit()
					}),
					func() tea.Msg {
						m.manager.StopContainer()
						return nil
					},
				)
			}

			if cmdText == "" {
				return m, nil
			}

			// Execute command async
			cmd := cmdText // capture for closure

			if cmd == "help" {
				m.output = append(m.output, "To quit the game, type 'exit'.")
				return m, nil
			}

			if cmd == "history" {
				for i, h := range m.history {
					m.output = append(m.output, fmt.Sprintf("%5d  %s", i+1, h))
				}
				return m, nil
			}

			return m, func() tea.Msg {
				out, err := m.manager.ExecuteCommand(cmd)
				return commandResultMsg{output: out, err: err}
			}

		case tea.KeyUp:
			if m.historyIdx > 0 {
				m.historyIdx--
				if m.historyIdx >= 0 && m.historyIdx < len(m.history) {
					m.input = m.history[m.historyIdx]
				}
			}
		case tea.KeyDown:
			if m.historyIdx < len(m.history) {
				m.historyIdx++
				if m.historyIdx == len(m.history) {
					m.input = ""
				} else {
					m.input = m.history[m.historyIdx]
				}
			}
		case tea.KeyBackspace:
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		case tea.KeyRunes:
			m.input += string(msg.Runes)
		case tea.KeySpace:
			m.input += " "
		}

	case questCheckMsg:
		if msg.passed {
			// Quest Complete Logic

			// Advance quest
			completedQuest := m.quests[msg.idx]
			m.output = append(m.output, "")

			// Quest complete notification remains in history
			headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
			m.output = append(m.output, headerStyle.Render(fmt.Sprintf(">>> QUEST COMPLETE! +%d XP <<<", completedQuest.XPReward)))

			// Success text in history
			successLines := strings.Split(completedQuest.SuccessText, "\n")
			m.output = append(m.output, successLines...)
			m.output = append(m.output, "")

			nextIdx := msg.idx + 1

			// Save Progress
			_ = game.SaveState(game.GameState{CurrentQuestID: nextIdx})

			if nextIdx < len(m.quests) {
				q := m.quests[nextIdx]

				// Show next quest info in Glitch box
				m.glitchText = fmt.Sprintf("(Next: %s)\n%s", q.Title, q.IntroText)
				m.currentQuestIdx = nextIdx
				m.output = append(m.output, fmt.Sprintf("--- QUEST %d: %s ---", q.ID, q.Title))

				// Run setup commands for the new quest
				return m, m.performQuestSetup(q)

			} else {
				m.glitchText = "You did it! All systems normal. <^.^>"
				m.currentQuestIdx = nextIdx
			}
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) startQuest(idx int) tea.Cmd {
	if idx >= len(m.quests) {
		m.glitchText = "You did it! All systems normal. <^.^>"
		return nil
	}
	m.currentQuestIdx = idx
	q := m.quests[idx]
	m.glitchText = q.IntroText
	m.output = append(m.output, fmt.Sprintf("--- QUEST %d: %s ---", q.ID, q.Title))

	return m.performQuestSetup(q)
}

func (m Model) performQuestSetup(q game.Quest) tea.Cmd {
	if len(q.SetupCommands) == 0 {
		return nil
	}
	return func() tea.Msg {
		for _, cmd := range q.SetupCommands {
			// Run setup commands silently
			// We use ExecuteValidation to run from root/home context as needed
			_, _ = m.manager.ExecuteValidation(cmd)
		}
		return nil
	}
}

func (m *Model) checkWinCondition() tea.Cmd {
	if m.currentQuestIdx >= len(m.quests) {
		return nil
	}

	q := m.quests[m.currentQuestIdx]

	return func() tea.Msg {
		// Validating state often requires running another command
		// This makes the flow complex because we can't easily return a Msg from here directly if we need to run an exec.
		// However, for simplicity, we will run the validation command blocks synchronously here
		// OR we dispatch a special validation msg.

		// BLOCKING CALL for validation (simple for prototype)
		checkPassed := false

		switch q.WinCondition.Type {
		case game.CommandOut:
			// "CommandOut" runs a command to validate game state.
			// e.g. "stat -c %a hut" should return "700"
			// We MUST use ExecuteValidation so it runs in a predictable context (/home/player)
			// independently of where the user has cd'd to.
			out, _ := m.manager.ExecuteValidation(q.WinCondition.Command)
			if strings.TrimSpace(out) == q.WinCondition.Expected {
				checkPassed = true
			}
		case game.DirExists:
			// check if dir exists using test -d, from ROOT context
			cmd := fmt.Sprintf("test -d %s && echo yes", q.WinCondition.Target)
			out, _ := m.manager.ExecuteValidation(cmd)
			if strings.TrimSpace(out) == "yes" {
				checkPassed = true
			}
		case game.FileExists:
			cmd := fmt.Sprintf("test -f %s && echo yes", q.WinCondition.Target)
			out, _ := m.manager.ExecuteValidation(cmd)
			if strings.TrimSpace(out) == "yes" {
				checkPassed = true
			}
		case game.FileContains:
			// check if file content contains string
			// We use grep in the container to check
			// safe because it's a validation command running in a controlled container
			// Escape single quotes for safety if needed, though basic check here:
			cmd := fmt.Sprintf("grep -q \"%s\" %s && echo yes", q.WinCondition.Content, q.WinCondition.Target)
			out, _ := m.manager.ExecuteValidation(cmd)
			if strings.TrimSpace(out) == "yes" {
				checkPassed = true
			}
		case game.UserOutputMatch:
			// Check if the *last* command output by the user matches the expectation
			// This is useful for "cat file" or "grep" where we want to see if they saw the right thing
			if strings.TrimSpace(m.lastOutput) == strings.TrimSpace(q.WinCondition.Expected) {
				checkPassed = true
			}
		case game.UserOutputContains:
			// Check if the *last* command output contains the expected string
			if strings.Contains(m.lastOutput, q.WinCondition.Expected) {
				checkPassed = true
			}
		case game.CurrentDirMatch:
			// Check if the current directory matches the target
			// The manager tracks CurrentDir
			// We need to handle relative vs absolute paths potentially?
			// For simplicity early game, target likely "hut" which implies "/home/player/hut"
			// But let's support both explicit absolute or relative to home.

			targetDir := q.WinCondition.Target
			// Normalize target
			if !strings.HasPrefix(targetDir, "/") {
				targetDir = "/home/player/" + targetDir
			}
			targetDir = strings.TrimSuffix(targetDir, "/")

			currentDir := strings.TrimSuffix(m.manager.CurrentDir, "/")

			if currentDir == targetDir {
				checkPassed = true
			}
		}

		return questCheckMsg{idx: m.currentQuestIdx, passed: checkPassed}
	}
}

type questCheckMsg struct {
	idx    int
	passed bool
}

// Need to handle the new msg type
// I need to update the Update function to handle questCompleteMsg
// Since I can't edit previous blocks, I will return the updated model content in full.
// Wait, I am writing the whole file. I need to insert the handler for questCompleteMsg in the Update function.
// I will rewrite the Update function below properly.

func (m Model) View() string {
	if !m.viewportReady {
		return "Initializing..."
	}

	// Styles
	screenStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Left, lipgloss.Top)

	// Layout components

	// 1. Header (Objective)
	objectiveText := "Load Quests..."
	if m.currentQuestIdx < len(m.quests) {
		objectiveText = m.quests[m.currentQuestIdx].Objective
	} else {
		objectiveText = "All Objectives Complete!"
	}

	header := lipgloss.NewStyle().
		Width(m.width).
		Height(1).
		Foreground(lipgloss.Color("#000000")).
		Background(lipgloss.Color("#AAAAAA")).
		PaddingLeft(1).
		Render(fmt.Sprintf("OBJECTIVE: %s", objectiveText))

	// 3. Glitch's Box (Bottom)
	// We render this FIRST to calculate remaining height for terminal

	// Process Glitch Text to colorize lines
	// We want System messages (Yellow) and Glitch (Green)
	lines := strings.Split(fmt.Sprintf("%s\n\n<'.'>", m.glitchText), "\n")
	var styledLines []string
	for _, line := range lines {
		styledLines = append(styledLines, styleLine(line))
	}
	styledGlitchText := strings.Join(styledLines, "\n")

	glitchBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00FF00")). // Green border for Glitch identity
		Padding(1).
		Width(m.width - 4). // Full width minus margins
		Render(styledGlitchText)

	// 4. Input Line
	// Pretty path: /home/player -> ~
	displayPath := m.manager.CurrentDir
	if strings.HasPrefix(displayPath, "/home/player") {
		displayPath = strings.Replace(displayPath, "/home/player", "~", 1)
	}

	inputLine := fmt.Sprintf("player@goblin:%s$ %s", displayPath, m.input)

	// Exit hint only for first quest
	if m.input == "" && m.currentQuestIdx == 0 {
		inputLine += lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Render(" (type 'exit' to quit)")
	}
	// Add blinking cursor
	if time.Now().UnixMilli()/500%2 == 0 {
		inputLine += "█"
	}

	// Calc heights
	// Header: 1
	// GlitchBox: lipgloss.Height(glitchBox)
	// Input: 1
	// Total Fixed = 1 + h(glitchBox) + 1

	totalFixedHeight := 1 + lipgloss.Height(glitchBox) + 1

	termHeight := m.height - totalFixedHeight
	if termHeight < 0 {
		termHeight = 0
	}

	// 2. Main Terminal Output
	// We need to account for wrapping to ensure we don't overflow the height
	contentWidth := m.width - 2 // -2 for horizontal padding
	if contentWidth < 1 {
		contentWidth = 1
	}

	wrapStyle := lipgloss.NewStyle().Width(contentWidth)
	var visibleLines []string

	// Iterate backwards through history to collect the most recent lines
	// accounting for line wrapping
	for i := len(m.output) - 1; i >= 0; i-- {
		// Style the line BEFORE wrapping to preserve ansi codes naturally?
		// No, styleLine adds ansi codes. wrapStyle handles them.
		lineContent := styleLine(m.output[i])

		// Render with wrapping
		rendered := wrapStyle.Render(lineContent)
		lines := strings.Split(rendered, "\n")

		// Prepend lines (visual top-to-bottom order for this block) to our accumulator
		visibleLines = append(lines, visibleLines...)

		// Stop if we have enough lines to fill the screen
		if len(visibleLines) >= termHeight {
			break
		}
	}

	// Truncate to exact height if we collected too many
	if len(visibleLines) > termHeight {
		visibleLines = visibleLines[len(visibleLines)-termHeight:]
	}

	mainTerm := lipgloss.NewStyle().
		Width(m.width).
		Height(termHeight).
		Padding(0, 1). // Horizontal padding
		Render(strings.Join(visibleLines, "\n"))

	return screenStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			mainTerm,
			glitchBox,
			inputLine,
		),
	)
}

func styleLine(text string) string {
	// If the line already has ansi codes (e.g. from SuccessText), we might want to skip or be careful.
	// Simple check: if it starts with [SYSTEM MESSAGE], color it Orange.
	if strings.Contains(text, "[SYSTEM MESSAGE]") {
		// Orange/Yellow
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500")).Render(text)
	}
	if strings.Contains(text, "<'.'>") {
		// Green
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render(text)
	}
	// Default: return as is (white/terminal default)
	return text
}
