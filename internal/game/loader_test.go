package game

import (
	"path/filepath"
	"regexp"
	"testing"
)

func loadRealQuests(t *testing.T) []Quest {
	t.Helper()
	// We point to the actual file for this integration test
	// In a strict unit test we'd use a temp file
	path := filepath.Join("..", "..", "quests", "quests.yaml")

	quests, err := LoadQuests(path)
	if err != nil {
		t.Fatalf("Failed to load quests: %v", err)
	}
	return quests
}

func TestLoadQuests(t *testing.T) {
	quests := loadRealQuests(t)

	if len(quests) == 0 {
		t.Errorf("Expected at least one quest, got 0")
	}

	if quests[0].ID != 1 {
		t.Errorf("Expected first quest ID to be 1, got %d", quests[0].ID)
	}
}

func TestQuestsHaveTeachingContent(t *testing.T) {
	quests := loadRealQuests(t)
	for i, q := range quests {
		if q.ID != i+1 {
			t.Errorf("quest at index %d has ID %d, want %d", i, q.ID, i+1)
		}
		if q.HardObjective == "" {
			t.Errorf("quest %d has no hard_objective", q.ID)
		}
		if q.HardObjective == q.Objective {
			t.Errorf("quest %d: hard_objective gives away the command", q.ID)
		}
		if i == len(quests)-1 {
			continue // the final 'exit' quest teaches nothing new
		}
		if len(q.Hints) == 0 {
			t.Errorf("quest %d has no hints", q.ID)
		}
		if q.Lesson == "" {
			t.Errorf("quest %d has no lesson", q.ID)
		}
	}
}

func TestUserOutputRegexPattern(t *testing.T) {
	re := regexp.MustCompile(`uid=\d+\(glitch\)`)
	if !re.MatchString("uid=1001(glitch) gid=1001(glitch) groups=1001(glitch)") {
		t.Error("id glitch output should match")
	}
	if re.MatchString("uid=1000(player) gid=1000(player) groups=1000(player),27(sudo)") {
		t.Error("id (for player) output should not match")
	}
}

func TestMistakePatterns(t *testing.T) {
	quests := loadRealQuests(t)
	byID := map[int]Quest{}
	for _, q := range quests {
		byID[q.ID] = q
	}

	matchesAny := func(q Quest, cmd string) bool {
		for _, m := range q.Mistakes {
			if m.Matches(cmd) {
				return true
			}
		}
		return false
	}

	cases := []struct {
		quest int
		cmd   string
		want  bool
	}{
		{3, "cd tmp", true},
		{3, "cd /tmp", false},
		{5, "mkdir .safe_house", true},
		{5, "mv safe_house .safe_house", false},
		{6, "touch cookie", true},
		{6, "touch .safe_house/cookie", false},
		{10, "sudo useradd glitch", true},
		{10, "useradd -m glitch", true}, // missing sudo
		{10, "sudo useradd -m glitch", false},
		{13, "ps | grep scanner_daemon", true},
		{13, "ps -e | grep scanner_daemon", false},
		{14, "kill scanner_daemon", true},
		{14, "kill 1234", false},
		{14, "killall scanner_daemon", false},
		{16, "grep CURE /usr/share/doc/data_dump.txt", true},
		{16, `grep "CURE-" /usr/share/doc/data_dump.txt`, true},
		{16, `grep "CURE-[0-9]{4}" /usr/share/doc/data_dump.txt`, true},
		{16, `grep -E "CURE-[0-9]{4}" /usr/share/doc/data_dump.txt`, false},
		{17, "id", true},
		{17, "id glitch", false},
		{18, "sudo usermod -G sudo glitch", true},
		{18, "sudo usermod -aG sudo glitch", false},
		{18, "sudo usermod -a -G sudo glitch", false},
		{19, "sudo chage -E 0 glitch", true},
		{19, "sudo chage -d 0 glitch", false},
		{22, `echo * * * * * date >> heartbeat.log | crontab -`, true},
		{22, `echo "* * * * * date >> heartbeat.log" | crontab -`, false},
		{23, "sudo tar -czf .safe_house backpack.img glitch.tar.gz", true},
		{23, "sudo tar -cf glitch.tar.gz .safe_house backpack.img", true},
		{24, "ssh-keygen", true},
		{24, `ssh-keygen -t rsa -f id_rsa -N ""`, false},
		{25, "ping gateway", true},
		{25, "ping -c 3 gateway", false},
		{27, "hostname", true},
		{27, "ssh -i id_rsa.pub player@gateway hostname", true},
		{27, "ssh -i id_rsa player@gateway hostname", false},
		{28, "scp -i id_rsa glitch.tar.gz player@gateway", true},
		{28, "scp -i id_rsa glitch.tar.gz player@gateway:~", false},
	}

	for _, c := range cases {
		q, ok := byID[c.quest]
		if !ok {
			t.Fatalf("no quest %d", c.quest)
		}
		if got := matchesAny(q, c.cmd); got != c.want {
			t.Errorf("quest %d, %q: mistake matched = %v, want %v", c.quest, c.cmd, got, c.want)
		}
	}
}
