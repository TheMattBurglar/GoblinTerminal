package game

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadQuests parses a YAML file containing a list of quests
func LoadQuests(filepath string) ([]Quest, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read quest file: %w", err)
	}

	var quests []Quest
	if err := yaml.Unmarshal(data, &quests); err != nil {
		return nil, fmt.Errorf("failed to parse quests YAML: %w", err)
	}

	return quests, nil
}
