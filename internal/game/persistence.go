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
	CurrentQuestID int                 `json:"current_quest_id"`
	MsgLog         []string            `json:"msg_log"` // Optional: save history? For now just quest ID is key.
	Stats          map[int]*QuestStats `json:"stats,omitempty"`
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
