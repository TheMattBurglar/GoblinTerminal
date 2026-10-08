package game

import (
	"fmt"
	"os"
	"regexp"

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

	for qi := range quests {
		q := &quests[qi]
		for mi := range q.Mistakes {
			re, err := regexp.Compile(q.Mistakes[mi].Pattern)
			if err != nil {
				return nil, fmt.Errorf("quest %d: invalid mistake pattern %q: %w", q.ID, q.Mistakes[mi].Pattern, err)
			}
			q.Mistakes[mi].re = re
		}
		if q.WinCondition.Type == UserOutputRegex {
			if _, err := regexp.Compile(q.WinCondition.Pattern); err != nil {
				return nil, fmt.Errorf("quest %d: invalid win pattern %q: %w", q.ID, q.WinCondition.Pattern, err)
			}
		}
	}

	return quests, nil
}
