package game

import (
	"regexp"
	"sort"
	"strings"
)

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
	// Answer is a question quest: the player investigates with normal
	// commands, then replies with 'answer <value>'. Passes if the value is
	// one of Answers, or a line printed by Command when the right answer
	// depends on what the player did earlier.
	Answer WinConditionType = "answer"
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
	// Answers are the accepted replies for Answer quests. Case, quotes and
	// spacing are ignored, and commas and spaces both separate list items.
	Answers []string `yaml:"answers,omitempty"`
	// Unordered accepts list answers in any order, e.g. group members.
	Unordered bool `yaml:"unordered,omitempty"`
}

var answerSeparators = regexp.MustCompile(`[\s,]+`)

// normalizeAnswer makes answers comparable: lower case, no surrounding
// quotes, list items separated by single commas.
func (wc *WinCondition) normalizeAnswer(a string) string {
	a = strings.ToLower(strings.TrimSpace(a))
	a = strings.Trim(a, `"'`)
	items := answerSeparators.Split(strings.Trim(a, " ,"), -1)
	if wc.Unordered {
		sort.Strings(items)
	}
	return strings.Join(items, ",")
}

// CheckAnswer reports whether the player's answer is one of Answers or of
// the extra accepted answers.
func (wc *WinCondition) CheckAnswer(answer string, accepted ...string) bool {
	given := wc.normalizeAnswer(answer)
	if given == "" {
		return false
	}
	for _, a := range append(accepted, wc.Answers...) {
		if wc.normalizeAnswer(a) == given {
			return true
		}
	}
	return false
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

// NewMistake builds a Mistake in code, for mistakes that apply to every quest.
func NewMistake(pattern, message string) Mistake {
	return Mistake{Pattern: pattern, Message: message, re: regexp.MustCompile(pattern)}
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

// IsQuestion reports whether the quest is completed with 'answer'.
func (q *Quest) IsQuestion() bool {
	return q.WinCondition.Type == Answer
}

// TotalHints is the number of hint levels, including the final reveal of
// the full command.
func (q *Quest) TotalHints() int {
	return len(q.Hints) + 1
}
