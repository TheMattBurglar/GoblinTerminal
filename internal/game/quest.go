package game

import "regexp"

// WinConditionType defines how we check if a quest is done
type WinConditionType string

const (
	DirExists          WinConditionType = "directory_exists"
	FileExists         WinConditionType = "file_exists"
	FileContains       WinConditionType = "file_content_contains"
	CommandOut         WinConditionType = "command_output_matches"
	UserOutputMatch    WinConditionType = "user_output_matches"
	UserOutputContains WinConditionType = "user_output_contains"
	UserOutputRegex    WinConditionType = "user_output_regex"
	// UserOutputHasResult runs Command in the container and passes if every
	// line it prints appears (as a whole word) in the user's last output.
	// Useful when the expected value is dynamic, e.g. a PID.
	UserOutputHasResult WinConditionType = "user_output_contains_result"
	CurrentDirMatch     WinConditionType = "current_working_directory"
	Custom              WinConditionType = "custom_check"
)

// WinCondition defines the criteria for completing a quest
type WinCondition struct {
	Type     WinConditionType `yaml:"type"`
	Target   string           `yaml:"target"`
	Content  string           `yaml:"content,omitempty"`
	Command  string           `yaml:"command,omitempty"`
	Expected string           `yaml:"expected_output,omitempty"`
	Pattern  string           `yaml:"pattern,omitempty"`
	// RejectOutput fails user-output checks if any of these strings appear,
	// e.g. to make sure the player filtered a file instead of cat-ing it.
	RejectOutput []string `yaml:"reject_output,omitempty"`
}

// Mistake is a common wrong approach to a quest. When the player's command
// matches Pattern and the quest isn't completed, Message is shown to explain
// what went wrong. Always shows it even if the command completed the quest,
// for approaches that work here but are risky in general.
type Mistake struct {
	Pattern string `yaml:"pattern"`
	Message string `yaml:"message"`
	Always  bool   `yaml:"always,omitempty"`

	re *regexp.Regexp
}

// Matches reports whether the command matches this mistake's pattern.
func (m *Mistake) Matches(command string) bool {
	return m.re != nil && m.re.MatchString(command)
}

// Quest represents a single level/objective in the game
type Quest struct {
	ID        int    `yaml:"id"`
	Title     string `yaml:"title"`
	IntroText string `yaml:"intro_text"`
	// Objective is the exact command, revealed as the final hint.
	Objective string `yaml:"objective"`
	// HardObjective describes the task without giving away the command.
	// It is what the objective bar shows by default.
	HardObjective string `yaml:"hard_objective"`
	// Hints are revealed one at a time by the 'hint' command, from vague to
	// specific. After the last one, Objective is shown.
	Hints []string `yaml:"hints,omitempty"`
	// Lesson is shown after the quest is completed and explains the command.
	Lesson        string       `yaml:"lesson,omitempty"`
	Mistakes      []Mistake    `yaml:"mistakes,omitempty"`
	WinCondition  WinCondition `yaml:"win_condition"`
	SuccessText   string       `yaml:"success_text"`
	XPReward      int          `yaml:"xp_reward"`
	Environment   string       `yaml:"environment"` // "local" or "container_image:..."
	SetupCommands []string     `yaml:"setup_commands,omitempty"`
}

// TotalHints is the number of hint levels, including the final reveal of
// the full command.
func (q *Quest) TotalHints() int {
	return len(q.Hints) + 1
}
