package game

// WinConditionType defines how we check if a quest is done
type WinConditionType string

const (
	DirExists          WinConditionType = "directory_exists"
	FileExists         WinConditionType = "file_exists"
	FileContains       WinConditionType = "file_content_contains"
	CommandOut         WinConditionType = "command_output_matches"
	UserOutputMatch    WinConditionType = "user_output_matches"
	UserOutputContains WinConditionType = "user_output_contains"
	CurrentDirMatch    WinConditionType = "current_working_directory"
	Custom             WinConditionType = "custom_check"
)

// WinCondition defines the criteria for completing a quest
type WinCondition struct {
	Type     WinConditionType `yaml:"type"`
	Target   string           `yaml:"target"`
	Content  string           `yaml:"content,omitempty"`
	Command  string           `yaml:"command,omitempty"`
	Expected string           `yaml:"expected_output,omitempty"`
}

// Quest represents a single level/objective in the game
type Quest struct {
	ID            int          `yaml:"id"`
	Title         string       `yaml:"title"`
	IntroText     string       `yaml:"intro_text"`
	Objective     string       `yaml:"objective"`
	WinCondition  WinCondition `yaml:"win_condition"`
	SuccessText   string       `yaml:"success_text"`
	XPReward      int          `yaml:"xp_reward"`
	Environment   string       `yaml:"environment"` // "local" or "container_image:..."
	SetupCommands []string     `yaml:"setup_commands,omitempty"`
}
