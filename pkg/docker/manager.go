package docker

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Manager handles the lifecycle of the game container
type Manager struct {
	ImageName     string
	ContainerName string
	GatewayName   string // New: Gateway container name
	NetworkName   string // New: Custom network name
	Runtime       string // "docker" or "podman"
	CurrentDir    string // Tracks the current working directory in the container
}

// NewManager creates a new container manager
func NewManager(imageName, containerName string) (*Manager, error) {
	// Check for container runtime
	runtime := "docker"
	if _, err := exec.LookPath("podman"); err == nil {
		runtime = "podman"
		fmt.Printf("Container Manager: Using Podman runtime.\n")
	} else if _, err := exec.LookPath("docker"); err != nil {
		return nil, fmt.Errorf("neither podman nor docker found in PATH")
	} else {
		fmt.Printf("Container Manager: Using Docker runtime.\n")
	}

	return &Manager{
		ImageName:     imageName,
		ContainerName: containerName,
		GatewayName:   containerName + "_gateway",
		NetworkName:   "goblin_net",
		Runtime:       runtime,
		CurrentDir:    "/home/player", // Default start dir
	}, nil
}

// BuildImage builds the docker image from the Dockerfile
func (m *Manager) BuildImage() error {
	cmd := exec.Command(m.Runtime, "build", "-t", m.ImageName, ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to build image: %v\nOutput: %s", err, string(output))
	}
	return nil
}

// EnsureNetwork creates the custom network if it doesn't exist
func (m *Manager) EnsureNetwork() error {
	// Check if network exists
	checkCmd := exec.Command(m.Runtime, "network", "inspect", m.NetworkName)
	if err := checkCmd.Run(); err == nil {
		return nil // Network exists
	}

	// Create network with specific subnet
	// docker network create --subnet=10.10.10.0/24 goblin_net
	cmd := exec.Command(m.Runtime, "network", "create", "--subnet=10.10.10.0/24", m.NetworkName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create network: %v\nOutput: %s", err, string(out))
	}
	return nil
}

// StartContainer starts the game containers (Gateway + Terminal)
func (m *Manager) StartContainer() error {
	// 1. Cleanup old containers
	m.StopContainer()

	// 2. Ensure Network
	if err := m.EnsureNetwork(); err != nil {
		return err
	}

	// 3. Start Gateway Container (The Target)
	// runs sshd
	// IP: 10.10.10.2
	// Needs to run as root (User 0) to bind port 22 and needs host keys generated
	gatewayCmd := exec.Command(m.Runtime, "run", "-d", "--rm",
		"--name", m.GatewayName,
		"--network", m.NetworkName,
		"--ip", "10.10.10.2",
		"--hostname", "gateway",
		"--user", "0",
		m.ImageName,
		"bash", "-c", "ssh-keygen -A && /usr/sbin/sshd -D")

	if out, err := gatewayCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to start gateway: %v\nOutput: %s", err, string(out))
	}

	// 4. Start Player Container (The Terminal)
	// IP: 10.10.10.3
	// Ensure local storage directory exists
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home dir: %v", err)
	}
	localPath := filepath.Join(homeDir, ".local", "share", "goblin-terminal", "fs")
	if err := os.MkdirAll(localPath, 0755); err != nil {
		return fmt.Errorf("failed to create local storage directory: %v", err)
	}
	if err := os.Chmod(localPath, 0777); err != nil {
		return fmt.Errorf("failed to chmod local storage directory: %v", err)
	}

	playerCmd := exec.Command(m.Runtime, "run", "-d", "--rm", "--init",
		"--cap-add=NET_RAW",
		"--name", m.ContainerName,
		"--network", m.NetworkName,
		"--ip", "10.10.10.3",
		"--hostname", "goblin",
		"-v", fmt.Sprintf("%s:/home/player:z", localPath),
		m.ImageName)

	if out, err := playerCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to start container: %v\nOutput: %s", err, string(out))
	}

	// Reset dir on start
	m.CurrentDir = "/home/player"
	return nil
}

// StopContainer stops and removes the containers
func (m *Manager) StopContainer() error {
	// Stop Player
	cmd := exec.Command(m.Runtime, "rm", "-f", m.ContainerName)
	_ = cmd.Run()

	// Stop Gateway
	cmd2 := exec.Command(m.Runtime, "rm", "-f", m.GatewayName)
	_ = cmd2.Run()

	return nil
}

// ErrTimeout is returned when a command is killed for running too long,
// usually because it is interactive and waiting for input.
var ErrTimeout = fmt.Errorf("command timed out")

// commandTimeout is how long a player command may run before being killed.
const commandTimeout = 5 * time.Second

// ExecuteCommand runs a command inside the container and returns stdout and
// stderr separately, so callers can validate stdout without stderr noise.
func (m *Manager) ExecuteCommand(command string) (string, string, error) {
	// Handle 'cd' specially
	trimmedCmd := strings.TrimSpace(command)
	if strings.HasPrefix(trimmedCmd, "cd ") || trimmedCmd == "cd" {
		target := "/home/player" // default cd
		if len(trimmedCmd) > 3 {
			target = strings.TrimSpace(trimmedCmd[3:])
		}

		// To safely change directory, we try to cd AND print pwd
		// We execute this from the CURRENT tracked directory
		// escaping is tricky so we construct a secure command

		// "cd <current> && cd <target> && pwd"
		fullCmd := fmt.Sprintf("cd %s && cd %s && pwd", m.CurrentDir, target)

		args := []string{"exec", m.ContainerName, "bash", "-c", fullCmd}
		cmd := exec.Command(m.Runtime, args...)

		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			// If cd fails, return the error (e.g. no such directory)
			errStr := stderr.String()
			if errStr == "" {
				errStr = "No such file or directory" // default generic
			}
			return "", "", fmt.Errorf("%s", errStr)
		}

		// Update persistent state
		newDir := strings.TrimSpace(out.String())
		if newDir != "" {
			m.CurrentDir = newDir
		}
		return "", "", nil // cd produces no output on success
	}

	// For normal commands, execute them in the current working directory
	// docker exec -w /current/path ...

	args := []string{"exec", "-w", m.CurrentDir, m.ContainerName, "bash", "-c", command}
	cmd := exec.Command(m.Runtime, args...)

	// Create a timer to kill command if it hangs
	timedOut := false
	timer := time.AfterFunc(commandTimeout, func() {
		if cmd.Process != nil {
			timedOut = true
			cmd.Process.Kill()
		}
	})
	defer timer.Stop()

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if timedOut {
		err = ErrTimeout
	}

	return out.String(), stderr.String(), err
}

// Files a terminal session leaves in the container: a transcript of
// everything it printed, and the shell's last working directory.
const (
	sessionLog = "/tmp/.goblin_session"
	sessionCwd = "/tmp/.goblin_cwd"
)

// TerminalCommand returns a command that runs one player command in the
// container with a real terminal, for programs that ask questions or draw
// the whole screen (nano, fdisk, passwd, top...). The output is recorded
// with script(1) so it can be shown and checked afterwards.
func (m *Manager) TerminalCommand(command string) *exec.Cmd {
	return m.terminalCmd(fmt.Sprintf("script -qefc %s %s", shellQuote(command), sessionLog))
}

// ShellCommand returns a command that opens an interactive bash shell in the
// container. Its working directory is saved at each prompt, so the game can
// pick up where the player left off.
func (m *Manager) ShellCommand() *exec.Cmd {
	// The home directory is a bind mount, so it has no .bashrc until the
	// player makes one. Fall back to Ubuntu's default for the usual prompt.
	shell := `if [ -f ~/.bashrc ]; then exec bash -i; else exec bash --rcfile /etc/skel/.bashrc -i; fi`
	banner := `printf '\033[1;32m<^.^> Goblin shell. Type exit to return to the game.\033[0m\n'`
	return m.terminalCmd(fmt.Sprintf("%s; script -qefc %s %s", banner, shellQuote(shell), sessionLog))
}

func (m *Manager) terminalCmd(script string) *exec.Cmd {
	_ = m.RunAsRoot(fmt.Sprintf("rm -f %s %s", sessionLog, sessionCwd))
	term := os.Getenv("TERM")
	if term == "" {
		term = "xterm-256color"
	}
	return exec.Command(m.Runtime, "exec", "-it", "-w", m.CurrentDir,
		"-e", "TERM="+term,
		"-e", "SHELL=/bin/bash",
		"-e", fmt.Sprintf("PROMPT_COMMAND=pwd > %s 2>/dev/null", sessionCwd),
		m.ContainerName, "bash", "-c", script)
}

// FinishSession returns the raw transcript of the last terminal session and
// moves CurrentDir to wherever its shell ended up.
func (m *Manager) FinishSession() string {
	if dir, _ := m.ExecuteValidation("cat " + sessionCwd); strings.TrimSpace(dir) != "" {
		m.CurrentDir = strings.TrimSpace(dir)
	}
	out, _ := m.ExecuteValidation("cat " + sessionLog)
	return out
}

// shellQuote quotes s as a single bash word.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// ExecuteValidation runs a command from the root directory to check win conditions
// This ensures game logic is consistent regardless of where the user is cd'd to
func (m *Manager) ExecuteValidation(command string) (string, error) {
	// similar to ExecuteCommand but forcing -w "/" or just raw exec
	// actually we probably want to run from /home/player or /
	// Given the game context "target: hut/bed.txt", running from /home/player seems correct base

	args := []string{"exec", "-w", "/home/player", m.ContainerName, "bash", "-c", command}
	cmd := exec.Command(m.Runtime, args...)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// validation checks might fail (exit 1), we still want the output usually
		return out.String(), nil
	}
	return out.String(), nil
}

// ResetStorage removes the persistent storage directory
func (m *Manager) ResetStorage() error {
	// First, ensure the game container is stopped so it doesn't hold locks
	_ = m.StopContainer()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home dir: %v", err)
	}
	localPath := filepath.Join(homeDir, ".local", "share", "goblin-terminal", "fs")

	// Check if exists
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return nil // Nothing to do
	}

	// Permission Fix:
	// Files created in the container might have restrictive permissions (like 700) or belong to root.
	// We use a temporary container to chmod everything so we can delete it.
	// We mount localPath to /clean_target
	args := []string{"run", "--rm",
		"-u", "0", // Run as root to override ownership/permissions
		"-v", fmt.Sprintf("%s:/clean_target:z", localPath),
		m.ImageName,
		"chmod", "-R", "777", "/clean_target",
	}

	cmd := exec.Command(m.Runtime, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		// Just log error but attempt local removal anyway
		fmt.Printf("Warning: failed to fix permissions via docker: %v\nOutput: %s\n", err, string(out))
	}

	// Remove all contents
	if err := os.RemoveAll(localPath); err != nil {
		return fmt.Errorf("failed to remove storage directory: %v", err)
	}
	return nil
}

// RestoreEnvironment ensures the container state matches the player's progress.
// This handles cases like re-creating the 'glitch' user if the container was recreated.
// done reports whether the quest with the given ID has been completed.
func (m *Manager) RestoreEnvironment(done func(questID int) bool) error {
	// Quest 10: Create glitch user
	if done(10) {
		// Check if user exists using RunAsRoot (which returns error on failure)
		// ExecuteValidation swallows errors so it returns nil error even if user is missing
		err := m.RunAsRoot("id glitch")
		if err != nil {
			// User missing, recreate
			// We use useradd with -m usually, but the quest just said 'useradd glitch'
			// However, for persistence validation, we should ensure it's usable.
			// The original command in quest 10: "sudo useradd glitch"
			// We execute as root via docker exec

			// Note: We don't have sudo inside the validation exec unless we user root
			// ExecuteValidation runs as player. We need root.
			// Currently ExecuteCommand runs as 'docker exec ...' which defaults to root unless -u is passed?
			// Wait, StartContainer runs as default image user (likely root or player depending on Dockerfile).
			// The Dockerfile likely sets USER player?
			// Checking Dockerfile would be good, but assuming we can exec as root.

			// We'll use a helper to run as root
			if err := m.RunAsRoot("useradd -m glitch"); err != nil {
				// if fails (maybe it thinks it exists?), ignore or log
				// assuming clean state if id glitch failed.
				return fmt.Errorf("failed to restore glitch user: %v", err)
			}
		}
	}

	// Quest 18: Add glitch to sudo
	if done(18) {
		// Check if glitch is sudoer
		out, _ := m.ExecuteValidation("groups glitch")
		if !strings.Contains(out, "sudo") {
			if err := m.RunAsRoot("usermod -aG sudo glitch"); err != nil {
				return fmt.Errorf("failed to restore glitch sudo access: %v", err)
			}
		}
	}

	// Quest 19: Force glitch's password to expire
	if done(19) {
		_ = m.RunAsRoot("chage -d 0 glitch")
	}

	// Ensure ownership of .safe_house if past Quest 11
	if done(11) {
		// Quest 11: "sudo chown glitch /home/player/.safe_house"
		if _, err := m.ExecuteValidation("test -d /home/player/.safe_house"); err == nil {
			_ = m.RunAsRoot("chown glitch /home/player/.safe_house")
		}
	}

	// Ensure permissions of .safe_house if past Quest 12
	if done(12) {
		// Quest 12: "sudo chmod 700 /home/player/.safe_house"
		if _, err := m.ExecuteValidation("test -d /home/player/.safe_house"); err == nil {
			_ = m.RunAsRoot("chmod 700 /home/player/.safe_house")
		}
	}

	return nil
}

// RunAsRoot executes a command as root in the container
func (m *Manager) RunAsRoot(command string) error {
	args := []string{"exec", "-u", "0", m.ContainerName, "bash", "-c", command}
	cmd := exec.Command(m.Runtime, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%v: %s", err, string(out))
	}
	return nil
}
