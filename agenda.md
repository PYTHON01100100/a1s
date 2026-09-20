# Development Agenda: a1s (Alibaba Cloud ECS TUI)

## 🎯 Project Overview
**a1s** is an interactive, terminal-based user interface (TUI) tool written in **Go (Golang)** designed to manage Alibaba Cloud resources efficiently. Inspired by tools like `g1c` (for Google Cloud) and `e1s` (for AWS ECS), it aims to deliver sub-millisecond management capabilities directly from the terminal, specifically optimizing for developers operating within the Saudi Arabia (Riyadh) region.

---

## 🚀 Core Features (The Agendas)

### 1. ECS Instance Lifecycle & Live Monitor
* **Real-Time Data Grid:** A dynamic dashboard displaying ECS instances, their public/private IP addresses, regions, zones, and current state.
* **Background Auto-Refresh:** Uses a non-blocking background routine via Go's `time.Ticker` to update state without freezing the UI.
* **Instant Keybindings:** Fast instance state control shortcuts:
  * `s` -> Start Instance
  * `p` -> Stop Instance
  * `r` -> Reboot Instance
  * `d` -> Terminate/Delete Instance (with safety confirmation prompt)

### 2. Native Session Manager (SSH Terminal Session)
* **Problem Statement:** Avoid high-latency and slow rendering caused by traditional VNC and Web Workbench tools inside the Saudi region.
* **Solution Architecture:** Leverage Alibaba Cloud's **Cloud Assistant (ecs-assist)** and the `StartTerminalSession` API.
* **TUI Workflow:**
  * When a user selects an instance and presses `Enter`, the TUI application layout suspends (`app.Suspend()`).
  * It initializes a low-latency interactive TTY session using the official `ali-instance-cli` binary or standard WebSockets.
  * Inputs and outputs are piped directly to `os.Stdin`, `os.Stdout`, and `os.Stderr`.
  * Exiting the terminal automatically resumes the main `a1s` UI grid (`app.Resume()`).

### 3. Smart Remote File Export (Zero-Port Copying)
Allows instant file extraction from the instance directly to the local development environment without establishing standard SSH (Port 22) connections or managing explicit SFTP clients. Uses Alibaba Cloud's `RunCommand` orchestration infrastructure.

* **Single File Export:** 
  * Prompt user for a specific remote path (e.g., `/var/www/app/config.yaml`).
  * Run a remote command to convert the file target to Base64 format.
  * Capture the execution response buffer asynchronously via `DescribeInvocationResults`.
  * Decode the text stream locally and save it directly to the designated client folder.
* **Bulk YAML/Project Export:** 
  * Prompt user for a remote root project directory path.
  * Execute a background shell command pipeline: find all `.yaml`/`.yml` configuration specs, pack them tightly into an archive, compress it, encode it to Base64, and output the stream:
    ```bash
    find /path/to/project -type f \( -name "*.yaml" -o -name "*.yml" \) -print0 | tar -cvzf /tmp/yaml_export.tar.gz --null -T - && cat /tmp/yaml_export.tar.gz | base64 && rm /tmp/yaml_export.tar.gz
    ```
  * Download the raw string, unpack/decode natively in Go, and export as a single unified local `.tar.gz` bundle.

---

## 🛠️ Tech Stack & Dependencies
* **Language Platform:** Go (Golang)
* **TUI Framework:** `github.com/gdamore/tview` or `github.com/charmbracelet/bubbletea`
* **Official SDK Provider:** `github.com/aliyun/alibaba-cloud-sdk-go/services/ecs`
* **Configuration Mapper:** `github.com/spf13/viper` (For decoding native Aliyun CLI configuration tokens located at `~/.aliyun/config.json`)
