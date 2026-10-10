package game

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// QuestStats records how the player got through a quest.
type QuestStats struct {
	Attempts  int  `json:"attempts"`   // commands run that didn't complete the quest
	HintsUsed int  `json:"hints_used"` // highest hint level revealed
	Completed bool `json:"completed"`
}

type GameState struct {
	// CurrentQuest is the ID of the quest in progress. Saves track IDs, not
	// positions, so quests can be inserted without breaking old saves.
	CurrentQuest int  `json:"current_quest,omitempty"`
	Complete     bool `json:"complete,omitempty"`
	// LegacyIndex is the old save format: the 0-based position of the
	// current quest, from when quest N was always at position N-1.
	LegacyIndex *int                `json:"current_quest_id,omitempty"`
	MsgLog      []string            `json:"msg_log"` // Optional: save history? For now just quest ID is key.
	Stats       map[int]*QuestStats `json:"stats,omitempty"`
}

// legacyQuestCount is how many quests existed when saves stored positions.
const legacyQuestCount = 30

// QuestIndex returns the position in quests to resume from. A finished game
// returns len(quests).
func (s *GameState) QuestIndex(quests []Quest) int {
	id := s.CurrentQuest
	if s.Complete {
		return len(quests)
	}
	if id == 0 && s.LegacyIndex != nil {
		if *s.LegacyIndex >= legacyQuestCount {
			return len(quests)
		}
		id = *s.LegacyIndex + 1
	}
	for i, q := range quests {
		if q.ID == id {
			return i
		}
	}
	return 0
}

// SetQuestIndex records the position of the current quest in quests.
func (s *GameState) SetQuestIndex(quests []Quest, idx int) {
	s.LegacyIndex = nil
	s.Complete = idx >= len(quests)
	s.CurrentQuest = 0
	if !s.Complete && idx >= 0 {
		s.CurrentQuest = quests[idx].ID
	}
}

// QuestStats returns the stats for a quest ID, creating them if needed.
func (s *GameState) QuestStats(id int) *QuestStats {
	if s.Stats == nil {
		s.Stats = map[int]*QuestStats{}
	}
	st, ok := s.Stats[id]
	if !ok {
		st = &QuestStats{}
		s.Stats[id] = st
	}
	return st
}

func GetSavePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	saveDir := filepath.Join(configDir, "goblin-terminal")
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(saveDir, "save.json"), nil
}

func SaveState(state GameState) error {
	path, err := GetSavePath()
	if err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	return encoder.Encode(state)
}

func LoadState() (GameState, error) {
	path, err := GetSavePath()
	if err != nil {
		return GameState{}, err
	}

	file, err := os.Open(path)
	if err != nil {
		return GameState{}, err
	}
	defer file.Close()

	var state GameState
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&state)
	return state, err
}

func ResetState() error {
	path, err := GetSavePath()
	if err != nil {
		return err
	}
	return os.Remove(path)
}
