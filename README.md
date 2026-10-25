# Goblin Terminal

**Goblin Terminal** is an interactive terminal-based training game designed to help you master Linux command-line skills.

It puts you in the role of a system administrator ("Wizard") investigating a mysterious digital entity named **Glitch**. Through a series of story-driven quests, you will learn and practice essential Linux commands in a real, isolated container environment.

## Purpose

I built this project to serve as a fun, interactive challenge *inspired by* the **Canonical Using Linux Terminal (C.U.L.T.)** exam. While not a replacement for official study materials, the quest objectives are guided by the real-world skills covered in that certification.

## Key Features

*   **Story Mode**: A narrative-driven campaign across 3 Acts.
*   **Real Environment**: Commands are executed in a real Linux container (Docker/Podman). No simulation quirks—just real Linux.
*   **Real Interactive Programs**: Editors, pagers and prompts (`nano`, `vim`, `less`, `man`, `top`, `fdisk`, `passwd`, `adduser`, `crontab -e`, `visudo`, `ssh` logins...) get your real terminal, just like on the exam. The game comes back and checks your quest when you quit them. Type `shell` for a full bash session in the container; `exit` returns to the game.
*   **Twin-Container Architecture**: In Act 3, the game spins up a second "Gateway" container, allowing you to practice real SSH, SCP, and network connectivity quests between isolated systems.
*   **Exam-Inspired Challenges**: The quests are designed around the objectives of the **Canonical Using Linux Terminal** exam, covering skills such as:
    *   File & Directory Management, find, redirection (`>`, `>>`, `2>/dev/null`) and sed
    *   Permissions (numeric and symbolic chmod, setuid/setgid/sticky, umask) & ownership
    *   Users, groups, sudo and password policy (useradd/usermod, chage, sudoers, pwquality, login.defs)
    *   Processes (ps/kill), logs (syslog, auth.log) and logrotate
    *   Disks: dd, partitioning (parted/fdisk), mkfs and fsck on disk images
    *   Archives (tar), regex (grep), cron, and SSH (keys, scp, sshd_config hardening)
    *   *Note: Some topics like disk partitioning are adapted for the container environment.*
*   **Question Quests**: Some quests ask you to *read and interpret* something (an `ls -l` string, `/etc/shadow`, a sudoers rule, a crontab, a log) and reply with `answer <value>`, like the exam's "identify" objectives.
*   **Learn, Don't Copy**: Each quest tells you *what* to do, not which command to type. Type `hint` for progressively more specific clues; the last one reveals the full command.
*   **Lessons**: After each quest, a "What You Learned" card explains the command, its common options, and related exam knowledge.
*   **Mistake Feedback**: Common wrong approaches (e.g. `usermod -G` without `-a`, an unquoted cron line) get an explanation of what went wrong.
*   **Man Pages**: The container has full man pages (allowed on the real exam), so you can practice looking things up: `man useradd | grep -i -A1 'create.*home'`.
*   **Quality of Life**: Command history (Up/Down), scrollback (PgUp/PgDn), `help`, `clear`, and per-quest stats (attempts and hints used) saved with your progress.
*   **Hard Mode**: Hints are disabled. Run with `--hard` or toggle with Ctrl+H. I recommend replaying in this mode to test your recall.

## Prerequisites

*   **Go** (1.23+)
*   **Container Runtime**: Docker or Podman (Podman is recommended on Fedora/RHEL).

## Installation & Usage

1.  **Clone the repository**:
    ```bash
    git clone https://github.com/yourusername/goblin-terminal.git
    cd goblin-terminal
    ```

2.  **Build the game**:
    ```bash
    go build -o goblin-terminal .
    ```

3.  **Run**:
    ```bash
    ./goblin-terminal
    ```
    *Note: The first run will build the necessary container image, which may take a minute.*

## Running the Tests

```bash
go test ./...
```

A full end-to-end walkthrough that plays every quest against real containers is opt-in:

```bash
docker build -t goblin-terminal:latest .
GOBLIN_INTEGRATION=1 go test ./internal/ui -run Walkthrough -v
```

It uses a temporary home and config directory, so it won't touch your saved game.

## License

This project is dual-licensed to separate the code from the creative content:

*   **Code**: The source code (Go, Dockerfile, Scripts) is licensed under the **GNU General Public License v3.0 (GPLv3)**. See [LICENSE](LICENSE).
*   **Content**: The story, characters, and quest text are licensed under the **Creative Commons Attribution-ShareAlike 4.0 International (CC BY-SA 4.0)**. See [CONTENT_LICENSE.md](CONTENT_LICENSE.md).

## Credits

Created by **TheMattBurglar** as a study aid for the C.U.L.T. exam.
