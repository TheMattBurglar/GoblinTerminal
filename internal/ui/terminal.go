package ui

import (
	"regexp"
	"strings"
)

// sudoPrefix matches an optional leading sudo, e.g. "sudo " or "sudo -u glitch ".
const sudoPrefix = `^\s*(sudo\s+(-u\s+\S+\s+)?)?`

// terminalCommands match programs that need a real terminal: they draw the
// whole screen, ask questions, or run until Ctrl+C. The game hands the real
// terminal over to them instead of running them in its own output pane.
var terminalCommands = []*regexp.Regexp{
	// Editors, pagers and live monitors
	regexp.MustCompile(sudoPrefix + `(nano|vi|vim|view|vimtutor|emacs|less|more|top|htop|watch|visudo|vipw|vigr|sudoedit)(\s|$)`),
	regexp.MustCompile(`^\s*sudo\s+-e\s`),
	regexp.MustCompile(sudoPrefix + `crontab\s+(-u\s+\S+\s+)?-e\b`),
	// man without a pipe or redirect opens the pager
	regexp.MustCompile(sudoPrefix + `man\s+[^|<>]+$`),
	// Programs that prompt for answers
	regexp.MustCompile(sudoPrefix + `passwd(\s+[a-z_][a-z0-9_-]*)?\s*$`),
	regexp.MustCompile(sudoPrefix + `(adduser|cfdisk|gdisk)\b`),
	regexp.MustCompile(sudoPrefix + `(chfn|chsh)(\s+[a-z_][a-z0-9_-]*)?\s*$`),
	regexp.MustCompile(sudoPrefix + `fdisk\s+[^-\s]\S*\s*$`),
	regexp.MustCompile(sudoPrefix + `parted(\s+[^-\s]\S*)?\s*$`),
	// Shells and logins
	regexp.MustCompile(`^\s*(su|login|bash|sh)(\s+-|\s+[a-z_][a-z0-9_-]*)*\s*$`),
	regexp.MustCompile(`^\s*sudo\s+(-i|-s|su|bash)\b`),
	// Commands that never finish on their own
	regexp.MustCompile(sudoPrefix + `(tail|journalctl)\s.*(-f|-F|--follow)\b`),
}

var (
	// fsckFlags are the options that let e2fsck run without asking questions.
	fsckFlags = regexp.MustCompile(`\s-\w*[npya]`)
	// keygenOther are ssh-keygen options for jobs other than making a key.
	keygenOther = regexp.MustCompile(`\s-[lyRFHeiL]`)
)

// needsTerminal reports whether cmd should run with a real terminal.
func needsTerminal(cmd string) bool {
	for _, re := range terminalCommands {
		if re.MatchString(cmd) {
			return true
		}
	}
	args := strings.Fields(cmd)
	if len(args) > 0 && args[0] == "sudo" {
		args = args[1:]
	}
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "e2fsck", "fsck", "fsck.ext2", "fsck.ext3", "fsck.ext4":
		// Without -n, -p, -y or -a, e2fsck asks before every repair.
		return !fsckFlags.MatchString(cmd)
	case "ssh-keygen":
		// Generating a key asks for a passphrase unless -N gives one.
		return !strings.Contains(cmd, "-N") && !keygenOther.MatchString(cmd)
	case "ssh":
		return sshIsInteractive(args[1:])
	case "ping":
		// Without -c, ping runs until Ctrl+C.
		return !strings.Contains(cmd, "-c")
	}
	return false
}

// sshArgOptions are ssh options that take a value, e.g. -i id_rsa.
const sshArgOptions = "BbcDEeFIiJLlmOoPpQRSWw"

// sshIsInteractive reports whether ssh args open a login shell: a host but
// no remote command.
func sshIsInteractive(args []string) bool {
	positional := 0
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") && len(a) > 1 {
			// "-i file" takes the next word, "-ifile" doesn't
			if len(a) == 2 && strings.ContainsRune(sshArgOptions, rune(a[1])) {
				i++
			}
			continue
		}
		positional++
	}
	return positional == 1
}

var (
	// altScreen matches what a full-screen program (nano, top, less...) drew
	// on the alternate screen. It's gone when the program exits, so it's
	// dropped from the transcript too.
	altScreen = regexp.MustCompile(`(?s)\x1b\[\?(?:1049|1047|47)h.*?(?:\x1b\[\?(?:1049|1047|47)l|$)`)
	// escapes matches terminal control sequences: colors, cursor moves,
	// window titles.
	escapes = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]|\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)|\x1b[()][0-9A-Za-z]|\x1b[@-_]`)
)

// cleanTranscript turns a script(1) recording into plain lines of text, as
// they looked on screen once the session ended.
func cleanTranscript(raw string) string {
	lines := strings.Split(raw, "\n")
	// script adds "Script started on ..." and "Script done on ..." lines.
	if len(lines) > 0 && strings.HasPrefix(lines[0], "Script started") {
		lines = lines[1:]
	}
	for len(lines) > 0 {
		last := strings.TrimSpace(lines[len(lines)-1])
		if last != "" && !strings.HasPrefix(last, "Script done") {
			break
		}
		lines = lines[:len(lines)-1]
	}
	text := altScreen.ReplaceAllString(strings.Join(lines, "\n"), "")
	text = escapes.ReplaceAllString(text, "")

	var out []string
	for _, line := range strings.Split(text, "\n") {
		// Terminals end lines with \r\n. Any other carriage return moves back
		// to the start of the line, so only what came after the last one shows.
		line = strings.TrimRight(line, "\r")
		if i := strings.LastIndex(line, "\r"); i >= 0 {
			line = line[i+1:]
		}
		out = append(out, applyBackspaces(line))
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n")
}

// applyBackspaces removes the characters that a backspace erased.
func applyBackspaces(line string) string {
	if !strings.ContainsRune(line, '\b') {
		return line
	}
	var buf []rune
	for _, r := range line {
		if r == '\b' {
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
			}
			continue
		}
		buf = append(buf, r)
	}
	return string(buf)
}
