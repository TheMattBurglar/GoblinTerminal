# Goblin Terminal

**Goblin Terminal** is an interactive terminal-based training game designed to help you master Linux command-line skills.

It puts you in the role of a system administrator ("Wizard") investigating a mysterious digital entity named **Glitch**. Through a series of story-driven quests, you will learn and practice essential Linux commands in a real, isolated container environment.

## Purpose

I built this project to serve as a fun, interactive challenge **inspired by** the **Canonical Using Linux Terminal (C.U.L.T.)** exam. While not a replacement for official study materials, the quest objectives are guided by the real-world skills covered in that certification.

## Key Features

*   **Story Mode**: A narrative-driven campaign across 3 Acts.
*   **Real Environment**: Commands are executed in a real Linux container (Docker/Podman). No simulation quirks—just real Linux.
*   **Twin-Container Architecture**: In Act 3, the game spins up a second "Gateway" container, allowing you to practice real SSH, SCP, and network connectivity quests between isolated systems.
*   **Exam-Inspired Challenges**: The quests are designed around the objectives of the **Canonical Using Linux Terminal** exam, covering skills such as:
    *   File & Directory Management
    *   Permissions (chmod/chown) & Users (useradd/sudo)
    *   Processes (ps/kill) & Logs (syslog)
    *   Archives (tar) & Regex (grep)
    *   *Note: Some topics like disk partitioning are adapted for the container environment.*
*   **Quality of Life**: Built-in command history (Up/Down arrows) and strict win-condition validation.
*   **Immediate Feedback**: The game validates your actions in real-time (e.g., checking if a file was actually created or permissions changed).

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

## License

This project is dual-licensed to separate the code from the creative content:

*   **Code**: The source code (Go, Dockerfile, Scripts) is licensed under the **GNU General Public License v3.0 (GPLv3)**. See [LICENSE](LICENSE).
*   **Content**: The story, characters, and quest text are licensed under the **Creative Commons Attribution-ShareAlike 4.0 International (CC BY-SA 4.0)**. See [CONTENT_LICENSE.md](CONTENT_LICENSE.md).

## Credits

Created by **TheMattBurglar** as a study aid for the C.U.L.T. exam.
