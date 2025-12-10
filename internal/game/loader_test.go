package game

import (
	"path/filepath"
	"testing"
)

func TestLoadQuests(t *testing.T) {
	// We point to the actual file for this integration test
	// In a strict unit test we'd use a temp file
	path := filepath.Join("..", "..", "quests", "quests.yaml")

	quests, err := LoadQuests(path)
	if err != nil {
		t.Fatalf("Failed to load quests: %v", err)
	}

	if len(quests) == 0 {
		t.Errorf("Expected at least one quest, got 0")
	}

	if quests[0].ID != 1 {
		t.Errorf("Expected first quest ID to be 1, got %d", quests[0].ID)
	}
}
