package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goblin-terminal/internal/game"
	"goblin-terminal/pkg/docker"
)

func loadQuests(t *testing.T) []game.Quest {
	t.Helper()
	quests, err := game.LoadQuests(filepath.Join("..", "..", "quests", "quests.yaml"))
	if err != nil {
		t.Fatalf("loading quests: %v", err)
	}
	return quests
}

func TestRevealHint(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // don't touch the real save file
	quests := loadQuests(t)
	m := NewModel(quests, nil, game.GameState{}, 9, false) // quest 10: useradd
	q := quests[9]

	for i := 1; i <= q.TotalHints(); i++ {
		m.revealHint()
		if m.hintLevel != i {
			t.Fatalf("after %d hints, hintLevel = %d", i, m.hintLevel)
		}
	}
	last := m.output[len(m.output)-1]
	if !strings.Contains(last, q.Objective) {
		t.Errorf("final hint should reveal the full command, got %q", last)
	}
	if got := m.state.QuestStats(q.ID).HintsUsed; got != q.TotalHints() {
		t.Errorf("HintsUsed = %d, want %d", got, q.TotalHints())
	}

	// Asking again doesn't go past the last hint
	m.revealHint()
	if m.hintLevel != q.TotalHints() {
		t.Errorf("hintLevel went past the last hint: %d", m.hintLevel)
	}
}

func TestRevealHintHardMode(t *testing.T) {
	m := NewModel(loadQuests(t), nil, game.GameState{}, 0, true)
	m.revealHint()
	if m.hintLevel != 0 {
		t.Errorf("hard mode should not reveal hints")
	}
}

func TestMistakeFeedbackAlways(t *testing.T) {
	quests := loadQuests(t)
	q := quests[17] // quest 18: usermod -aG
	if lines := mistakeFeedback(q, "sudo usermod -G sudo glitch", true); len(lines) == 0 {
		t.Error("'always' mistakes should show even when the quest passed")
	}
	q = quests[2] // quest 3: cd /tmp
	if lines := mistakeFeedback(q, "cd tmp", true); len(lines) != 0 {
		t.Error("regular mistakes should not show when the quest passed")
	}
	if lines := mistakeFeedback(q, "cd tmp", false); len(lines) == 0 {
		t.Error("regular mistakes should show when the quest failed")
	}
}

func TestIsInteractive(t *testing.T) {
	for cmd, want := range map[string]bool{
		"nano file":            true,
		"sudo visudo":          true,
		"crontab -e":           true,
		"sudo passwd glitch":   true,
		"passwd -e glitch":     false,
		"crontab -l":           false,
		"vimtutor_is_not_real": false,
		"ls":                   false,
	} {
		if got := isInteractive(cmd); got != want {
			t.Errorf("isInteractive(%q) = %v, want %v", cmd, got, want)
		}
	}
}

// TestWalkthrough plays the whole game against real containers. It needs the
// goblin-terminal:latest image to be built first, and is opt-in:
//
//	docker build -t goblin-terminal:latest . && GOBLIN_INTEGRATION=1 go test ./internal/ui -run Walkthrough -v
func TestWalkthrough(t *testing.T) {
	if os.Getenv("GOBLIN_INTEGRATION") == "" {
		t.Skip("set GOBLIN_INTEGRATION=1 to run against real containers")
	}

	// Isolate the save file and the persistent home directory from real games.
	home, err := os.MkdirTemp("", "goblin-walkthrough-")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	mgr, err := docker.NewManager("goblin-terminal:latest", "goblin-walkthrough")
	if err != nil {
		t.Skipf("no container runtime: %v", err)
	}
	if err := mgr.StartContainer(); err != nil {
		t.Fatalf("starting containers: %v", err)
	}
	t.Cleanup(func() {
		_ = mgr.ResetStorage() // stops containers and removes container-owned files
		_ = os.RemoveAll(home)
	})

	quests := loadQuests(t)
	m := NewModel(quests, mgr, game.GameState{}, 0, false)
	if setup := m.startQuest(0); setup != nil {
		setup()
	}

	// run executes a command through the same message flow as the UI and
	// reports whether it completed the current quest.
	run := func(cmd string) bool {
		t.Helper()
		stdout, stderr, err := mgr.ExecuteCommand(cmd)
		model, check := m.Update(commandResultMsg{cmd: cmd, stdout: stdout, stderr: stderr, err: err})
		m = model.(Model)
		result := check().(questCheckMsg)
		model, setup := m.Update(result)
		m = model.(Model)
		if setup != nil {
			setup()
		}
		if !result.passed {
			t.Logf("  %q did not pass; stdout=%q stderr=%q err=%v", cmd, stdout, stderr, err)
		}
		return result.passed
	}

	steps := []struct {
		quest  int
		wrong  []string // commands that must NOT complete the quest
		setup  []string // commands that may or may not complete it
		answer string   // command that must complete it
	}{
		{1, nil, nil, "pwd"},
		{2, nil, nil, "ls -la /tmp"},
		{3, []string{"cd tmp"}, nil, "cd /tmp"},
		{4, nil, nil, "mkdir safe_house"},
		{5, nil, nil, "mv safe_house .safe_house"},
		{6, []string{"touch cookie"}, nil, "touch .safe_house/cookie"},
		{7, nil, nil, "cp .safe_house/cookie .safe_house/cookie_backup"},
		{8, nil, nil, "mv .safe_house ~"},
		{9, nil, []string{"cd ~"}, "whoami"},
		{10, []string{"sudo useradd glitch"}, []string{"sudo userdel glitch"}, "sudo useradd -m glitch"},
		{11, nil, nil, "sudo chown glitch .safe_house"},
		{12, nil, nil, "sudo chmod u=rwx,go= .safe_house"},
		{13, nil, nil, "pgrep scanner_daemon"},
		{14, nil, nil, "killall scanner_daemon"},
		{15, nil, nil, "sudo tail /var/log/syslog"},
		{16, []string{"grep CURE /usr/share/doc/data_dump.txt", "cat /usr/share/doc/data_dump.txt"}, nil,
			"grep -n -E 'CURE-[0-9]{4}' /usr/share/doc/data_dump.txt"},
		{17, []string{"id"}, nil, "id glitch"},
		{18, nil, nil, "sudo gpasswd -a glitch sudo"},
		{19, nil, nil, "sudo chage -d 0 glitch"},
		{20, []string{"dd if=/dev/zero of=backpack.img bs=1M count=1"}, nil, "dd if=/dev/zero of=backpack.img bs=1M count=100"},
		{21, nil, nil, "mkfs.ext4 backpack.img"},
		{22, nil, nil, "echo '*/1 * * * * /usr/bin/date >> /home/player/heartbeat.log' | crontab -"},
		{23, []string{"sudo tar -cf glitch.tar.gz .safe_house backpack.img"}, nil, "sudo tar -czf glitch.tar.gz .safe_house backpack.img"},
		{24, nil, nil, `ssh-keygen -t rsa -f id_rsa -N ""`},
		{25, nil, nil, "ping -c 3 gateway"},
		{26, nil, nil, "ssh-copy-id -i id_rsa.pub player@gateway"},
		{27, []string{"hostname"}, nil, "ssh -i id_rsa player@gateway hostname"},
		{28, nil, nil, "scp -i id_rsa glitch.tar.gz player@gateway:"},
		{29, nil, nil, "rm glitch.tar.gz"},
	}

	for _, s := range steps {
		if got := quests[m.currentQuestIdx].ID; got != s.quest {
			t.Fatalf("expected to be on quest %d, but on quest %d", s.quest, got)
		}
		for _, cmd := range s.wrong {
			if run(cmd) {
				t.Fatalf("quest %d: %q should not have completed the quest", s.quest, cmd)
			}
		}
		for _, cmd := range s.setup {
			run(cmd)
		}
		if !run(s.answer) {
			t.Fatalf("quest %d: %q should have completed the quest", s.quest, s.answer)
		}
	}

	if got := quests[m.currentQuestIdx].ID; got != 30 {
		t.Errorf("expected to finish on quest 30, got %d", got)
	}
}
