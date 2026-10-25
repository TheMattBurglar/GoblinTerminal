package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goblin-terminal/internal/game"
	"goblin-terminal/pkg/docker"

	tea "github.com/charmbracelet/bubbletea"
)

func loadQuests(t *testing.T) []game.Quest {
	t.Helper()
	quests, err := game.LoadQuests(filepath.Join("..", "..", "quests", "quests.yaml"))
	if err != nil {
		t.Fatalf("loading quests: %v", err)
	}
	return quests
}

// questIndex returns the position of the quest with this ID.
func questIndex(t *testing.T, quests []game.Quest, id int) int {
	t.Helper()
	for i, q := range quests {
		if q.ID == id {
			return i
		}
	}
	t.Fatalf("no quest %d", id)
	return -1
}

func TestRevealHint(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // don't touch the real save file
	quests := loadQuests(t)
	idx := questIndex(t, quests, 10) // useradd
	m := NewModel(quests, nil, game.GameState{}, idx, false)
	q := quests[idx]

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
	q := quests[questIndex(t, quests, 18)] // usermod -aG
	if lines := mistakeFeedback(q, "sudo usermod -G sudo glitch", true); len(lines) == 0 {
		t.Error("'always' mistakes should show even when the quest passed")
	}
	q = quests[questIndex(t, quests, 3)] // cd /tmp
	if lines := mistakeFeedback(q, "cd tmp", true); len(lines) != 0 {
		t.Error("regular mistakes should not show when the quest passed")
	}
	if lines := mistakeFeedback(q, "cd tmp", false); len(lines) == 0 {
		t.Error("regular mistakes should show when the quest failed")
	}

	q = quests[questIndex(t, quests, 10)] // useradd
	if lines := mistakeFeedback(q, "sudo useradd -m Glitch", false); len(lines) == 0 {
		t.Error("a capitalized user name should get the case-sensitivity tip")
	}
	if lines := mistakeFeedback(q, "sudo useradd -m glitch", false); len(lines) != 0 {
		t.Errorf("lower-case glitch should not get a tip, got %q", lines)
	}
}

func TestSubmitAnswer(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	quests := loadQuests(t)

	// 'answer' on a command quest is refused without counting an attempt.
	m := NewModel(quests, nil, game.GameState{}, 0, false)
	if cmd := m.submitAnswer("answer /home/player"); cmd != nil {
		t.Error("answer should not be checked on a command quest")
	}

	idx := questIndex(t, quests, 31)
	m = NewModel(quests, nil, game.GameState{}, idx, false)
	if cmd := m.submitAnswer("answer"); cmd != nil {
		t.Error("an empty answer should not be checked")
	}

	submit := func(cmd string) questCheckMsg {
		t.Helper()
		result := m.submitAnswer(cmd)().(questCheckMsg)
		model, _ := m.Update(result)
		m = model.(Model)
		return result
	}
	if submit("answer 457").passed {
		t.Fatal("wrong answer passed")
	}
	if got := m.state.QuestStats(31).Attempts; got != 1 {
		t.Errorf("wrong answer should count as an attempt, got %d", got)
	}
	if !submit("answer 754").passed {
		t.Fatal("right answer failed")
	}
	if m.currentQuestIdx != idx+1 {
		t.Errorf("should advance to the next quest, at %d", m.currentQuestIdx)
	}
	if saved, err := game.LoadState(); err != nil || saved.CurrentQuest != quests[idx+1].ID {
		t.Errorf("save should record the next quest's ID, got %+v (%v)", saved, err)
	}
}

func TestNeedsTerminal(t *testing.T) {
	for cmd, want := range map[string]bool{
		"nano file":                               true,
		"sudo visudo":                             true,
		"sudo -e /etc/hosts":                      true,
		"crontab -e":                              true,
		"sudo passwd glitch":                      true,
		"sudo adduser glitch":                     true,
		"fdisk map.img":                           true,
		"parted map.img":                          true,
		"man useradd":                             true,
		"e2fsck -f backpack.img":                  true,
		"ssh-keygen -t rsa":                       true,
		"ssh -i id_rsa player@gateway":            true,
		"su - glitch":                             true,
		"sudo -i":                                 true,
		"ping gateway":                            true,
		"sudo tail -f /var/log/syslog":            true,
		"passwd -e glitch":                        false,
		"crontab -l":                              false,
		"vimtutor_is_not_real":                    false,
		"ls":                                      false,
		"man useradd | grep -i home":              false,
		"fdisk -l map.img":                        false,
		"parted -s map.img mklabel gpt":           false,
		"parted map.img print":                    false,
		"e2fsck -fn backpack.img":                 false,
		`ssh-keygen -t rsa -f id_rsa -N ""`:       false,
		"ssh -i id_rsa player@gateway hostname":   false,
		"ping -c 3 gateway":                       false,
		"sudo tail /var/log/syslog":               false,
		"bash script.sh":                          false,
		"sudo adduser --disabled-password glitch": true,
	} {
		if got := needsTerminal(cmd); got != want {
			t.Errorf("needsTerminal(%q) = %v, want %v", cmd, got, want)
		}
	}
}

func TestCleanTranscript(t *testing.T) {
	raw := "Script started on 2026-10-25 [COMMAND=\"x\"]\n" +
		"\x1b[?2004hplayer@goblin:~$ ech\b \bho \x1b[01;32mhi\x1b[0m\r\n" +
		"hi\r\n" +
		"\x1b[?1049h\x1b[Hnano screen\r\nmore nano\x1b[?1049l" +
		"progress 10%\rprogress 100%\r\n" +
		"\r\n" +
		"Script done on 2026-10-25 [COMMAND_EXIT_CODE=\"0\"]\n"
	want := "player@goblin:~$ echo hi\nhi\nprogress 100%"
	if got := cleanTranscript(raw); got != want {
		t.Errorf("cleanTranscript() =\n%q\nwant\n%q", got, want)
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
		var check tea.Cmd
		var stdout, stderr string
		var err error
		if strings.HasPrefix(cmd, "answer ") {
			check = m.submitAnswer(cmd) // handled by the game, not the shell
		} else {
			stdout, stderr, err = mgr.ExecuteCommand(cmd)
			var model tea.Model
			model, check = m.Update(commandResultMsg{cmd: cmd, stdout: stdout, stderr: stderr, err: err})
			m = model.(Model)
		}
		if check == nil {
			return false // investigating during a question quest
		}
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
		{31, []string{"answer 457"}, []string{"ls -l /opt/eye"}, "answer 754"},
		{38, nil, nil, "sudo chmod a-x /opt/eye/report.sh"},
		{39, []string{"chmod +t /srv/dropbox"}, nil, "sudo chmod +t /srv/dropbox"},
		{40, []string{"answer 750"}, []string{"umask 027; touch umask_test; stat -c %a umask_test"}, "answer 640"},
		{32, []string{"answer 10.10.10.3"}, []string{"sudo grep 'Failed password' /var/log/auth.log"}, "answer 203.0.113.66"},
		{41, []string{"grep 'Failed password' /var/log/auth.log > evidence.txt"}, nil, "sudo grep 'Failed password' /var/log/auth.log > evidence.txt"},
		{42, []string{"date > evidence.txt"}, []string{"sudo grep 'Failed password' /var/log/auth.log > evidence.txt"}, "date >> evidence.txt"},
		{43, []string{"find / -name beacon 2>/dev/null"}, nil, "find / -name beacon.key 2>/dev/null"},
		{17, []string{"id"}, nil, "id glitch"},
		{33, nil, []string{"grep glitch /etc/passwd"}, "answer /bin/sh"},
		{18, nil, nil, "sudo gpasswd -a glitch sudo"},
		{34, nil, []string{"getent group sudo"}, "answer glitch, player, ubuntu"},
		{35, []string{"answer hunter"}, []string{"sudo cat /etc/sudoers.d/eye_wardens"}, "answer eye"},
		{44, []string{"find / -perm 4000 2>/dev/null", "find / -name eye_backdoor 2>/dev/null"}, nil, "find / -perm -4000 -type f 2>/dev/null"},
		{45, []string{"chmod u-s /usr/local/bin/eye_backdoor"}, nil, "sudo chmod u-s /usr/local/bin/eye_backdoor"},
		{19, nil, nil, "sudo chage -d 0 glitch"},
		{36, nil, []string{"sudo grep glitch /etc/shadow"}, "answer !"},
		{46, []string{"sed 's/^# minlen = 8/minlen = 12/' /etc/security/pwquality.conf"}, nil, "sudo sed -i 's/^# minlen = 8/minlen = 12/' /etc/security/pwquality.conf"},
		{47, nil, nil, "sudo sed -i 's/^PASS_MAX_DAYS.*/PASS_MAX_DAYS\t90/' /etc/login.defs"},
		{20, []string{"dd if=/dev/zero of=backpack.img bs=1M count=1"}, nil, "dd if=/dev/zero of=backpack.img bs=1M count=100"},
		{21, nil, nil, "mkfs.ext4 backpack.img"},
		{48, []string{"e2fsck backpack.img", "e2fsck -n backpack.img"}, nil, "e2fsck -fn backpack.img"},
		{49, []string{"parted -s map.img mklabel msdos mkpart primary ext4 1MiB 100%"}, nil, "parted -s map.img mklabel gpt mkpart map ext4 1MiB 100%"},
		{50, []string{"answer 2047"}, []string{"fdisk -l map.img"}, "answer 2048"},
		{22, nil, nil, "echo '*/1 * * * * /usr/bin/date >> /home/player/heartbeat.log' | crontab -"},
		{37, nil, []string{"cat /etc/cron.d/eye_maintenance"}, "answer Sunday"},
		{51, []string{"answer weekly"}, []string{"cat /etc/logrotate.d/eye"}, "answer 4"},
		{52, []string{"sudo logrotate -d /etc/logrotate.d/eye"}, nil, "sudo logrotate -f /etc/logrotate.d/eye"},
		{23, []string{"sudo tar -cf glitch.tar.gz .safe_house backpack.img map.img", "sudo tar -czf glitch.tar.gz .safe_house backpack.img"}, nil,
			"sudo tar -czf glitch.tar.gz .safe_house backpack.img map.img"},
		{24, nil, nil, `ssh-keygen -t rsa -f id_rsa -N ""`},
		{25, nil, nil, "ping -c 3 gateway"},
		{26, nil, nil, "ssh-copy-id -i id_rsa.pub player@gateway"},
		{27, []string{"hostname"}, nil, "ssh -i id_rsa player@gateway hostname"},
		{53, []string{"echo 'PasswordAuthentication no' | sudo tee -a /etc/ssh/sshd_config"}, []string{"sudo sed -i '$d' /etc/ssh/sshd_config"},
			"sudo sed -i 's/^PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config"},
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
