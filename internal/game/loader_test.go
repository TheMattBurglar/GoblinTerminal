package game

import (
	"path/filepath"
	"regexp"
	"strings"
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
		{12, "sudo chmod 700 /home/glitch", true},
		{12, "sudo chmod 700 /home/player/.safe_house", false},
		{13, "ps | grep scanner_daemon", true},
		{13, "ps -e | grep scanner_daemon", false},
		{14, "kill scanner_daemon", true},
		{14, "kill 1234", false},
		{14, "killall scanner_daemon", false},
		{16, "grep CURE /usr/share/doc/data_dump.txt", true},
		{16, "cat /usr/share/doc/data_dump.txt | grep CURE", true},
		{16, "cat /usr/share/doc/data_dump.txt | grep -E 'CURE-[0-9]{4}'", true}, // side note only
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
		{31, "answer 457", true},
		{31, "answer rwxr-xr--", true},
		{31, "answer 744", true},
		{32, "answer 10.10.10.3", true},
		{33, "answer /bin/bash", true},
		{33, "answer /home/glitch", true},
		{35, "answer %wardens", true},
		{35, "answer hunter", true},
		{36, "answer x", true},
		{36, "answer 0", true},
		{37, "answer monday", true},
		{37, "answer 30", true},
		{38, "sudo chmod 644 /opt/eye/report.sh", true},
		{38, "sudo chmod a-x /usr/share/doc/data_dump.txt", true},
		{39, "sudo chmod 4777 /srv/dropbox", true},
		{39, "sudo chmod 1777 /srv/dropbox", false},
		{40, "answer 750", true},
		{42, "date > evidence.txt", true},
		{42, "date >> evidence.txt", false},
		{43, "find / -name beacon.key", true},
		{43, "find / -name beacon.key 2>/dev/null", false},
		{43, "find / -name beacon 2>/dev/null", true},
		{44, "find / -perm 4000 2>/dev/null", true},
		{44, "find / -perm -4000 -type f 2>/dev/null", false},
		{46, "sed 's/^# minlen = 8/minlen = 12/' /etc/security/pwquality.conf", true},
		{46, "sudo sed -i 's/^# minlen = 8/minlen = 12/' /etc/security/pwquality.conf", true}, // the silent-sed tip; only shown if the quest failed
		{47, "sudo sed -i 's/PASS_MAX_DAYS.*/PASS_MAX_DAYS 90/' /etc/login.defs", true},
		{47, "sudo sed -i 's/^MAX_PASS_DAYS    99999/MAX_PASS_DAYS    90/' /etc/login.defs", true},
		{47, "sudo sed -i 's/^PASS_MAX_DAYS    99999/PASS_MAX_DAYS    90/' /etc/login.defs", true},
		{48, "fsck backpack.img", true},
		{48, "e2fsck -fn backpack.img", true}, // only shown if it failed
		{49, "fdisk map.img", true},
		{49, "parted map.img mklabel gpt", true},
		{49, "parted -s map.img mklabel gpt mkpart map ext4 1MiB 100%", false},
		{52, "sudo logrotate -d /etc/logrotate.d/eye", true},
		{52, "sudo logrotate -f /etc/logrotate.d/eye", false},
		{53, "echo 'PasswordAuthentication no' | sudo tee -a /etc/ssh/sshd_config", true},
		{53, "sudo sed -i 's/^PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config", true}, // the silent-sed tip; only shown if the quest failed
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

func TestQuestionAnswers(t *testing.T) {
	quests := loadRealQuests(t)
	byID := map[int]Quest{}
	for _, q := range quests {
		byID[q.ID] = q
	}

	cases := []struct {
		quest  int
		answer string
		want   bool
	}{
		{31, "754", true},
		{31, " 0754 ", true},
		{31, "457", false},
		{31, "", false},
		{32, "203.0.113.66", true},
		{32, "10.10.10.3", false},
		{33, "/bin/sh", true},
		{33, "SH", true},
		{33, "/bin/bash", false},
		{34, "ubuntu,player,glitch", true},
		{34, "glitch, player, ubuntu", true},
		{34, "player glitch ubuntu", true},
		{34, "player,glitch", false},
		{35, "eye", true},
		{35, "'eye'", true},
		{35, "hunter", false},
		{36, "!", true},
		{36, "x", false},
		{37, "Sunday", true},
		{37, "sun", true},
		{37, "7", true},
		{37, "monday", false},
		{40, "640", true},
		{40, "rw-r-----", true},
		{40, "750", false},
		{51, "4", true},
		{51, "weekly", false},
	}

	for _, c := range cases {
		q, ok := byID[c.quest]
		if !ok {
			t.Fatalf("no quest %d", c.quest)
		}
		if !q.IsQuestion() {
			t.Fatalf("quest %d is not a question", c.quest)
		}
		if got := q.WinCondition.CheckAnswer(c.answer); got != c.want {
			t.Errorf("quest %d, answer %q: accepted = %v, want %v", c.quest, c.answer, got, c.want)
		}
	}

	// The objective (revealed by the last hint) must itself be accepted.
	for _, q := range quests {
		if q.IsQuestion() && q.WinCondition.Command == "" && !q.WinCondition.CheckAnswer(strings.TrimPrefix(q.Objective, "answer ")) {
			t.Errorf("quest %d: objective %q is not an accepted answer", q.ID, q.Objective)
		}
	}
}

func TestQuestIndexMigration(t *testing.T) {
	quests := loadRealQuests(t)
	indexOf := func(id int) int {
		for i, q := range quests {
			if q.ID == id {
				return i
			}
		}
		t.Fatalf("no quest %d", id)
		return -1
	}
	legacy := func(i int) *int { return &i }

	cases := []struct {
		name  string
		state GameState
		want  int
	}{
		{"new game", GameState{}, 0},
		{"by ID", GameState{CurrentQuest: 18}, indexOf(18)},
		{"question by ID", GameState{CurrentQuest: 35}, indexOf(35)},
		{"complete", GameState{Complete: true}, len(quests)},
		// Old saves stored positions from when quest N was at position N-1.
		{"legacy start", GameState{LegacyIndex: legacy(0)}, 0},
		{"legacy quest 18", GameState{LegacyIndex: legacy(17)}, indexOf(18)},
		{"legacy finished", GameState{LegacyIndex: legacy(30)}, len(quests)},
		{"unknown ID", GameState{CurrentQuest: 999}, 0},
	}
	for _, c := range cases {
		if got := c.state.QuestIndex(quests); got != c.want {
			t.Errorf("%s: QuestIndex = %d, want %d", c.name, got, c.want)
		}
	}

	// Saving converts to the ID format and drops the legacy field.
	s := GameState{LegacyIndex: legacy(17)}
	s.SetQuestIndex(quests, s.QuestIndex(quests))
	if s.LegacyIndex != nil || s.CurrentQuest != 18 || s.Complete {
		t.Errorf("after SetQuestIndex: %+v", s)
	}
	s.SetQuestIndex(quests, len(quests))
	if !s.Complete || s.QuestIndex(quests) != len(quests) {
		t.Errorf("finished game should stay finished: %+v", s)
	}
}
