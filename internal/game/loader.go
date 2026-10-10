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

	seen := map[int]bool{}
	for qi := range quests {
		q := &quests[qi]
		if q.ID <= 0 || seen[q.ID] {
			return nil, fmt.Errorf("quest %q: ID %d is missing or duplicated", q.Title, q.ID)
		}
		seen[q.ID] = true
		if q.IsQuestion() && len(q.WinCondition.Answers) == 0 && q.WinCondition.Command == "" {
			return nil, fmt.Errorf("quest %d: answer quest has no answers", q.ID)
		}
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
