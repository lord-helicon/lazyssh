<div align="center">
  <img src="./docs/logo.png" alt="lazyssh logo" width="600" height="600"/>
</div>

---

Lazyssh is a terminal-based, interactive SSH manager inspired by tools like lazydocker and k9s — but built for managing your fleet of servers directly from your terminal.
<br/>
With lazyssh, you can quickly navigate, connect to, and manage any server defined in your `~/.ssh/config`. No more remembering IP addresses or long ssh commands — just a clean, keyboard-driven UI.

> **About this fork:** this is a maintained fork of [Adembc/lazyssh](https://github.com/Adembc/lazyssh), which hasn't merged changes since v0.3.0. It adds fixes for how `~/.ssh/config` is written (symlinked configs, `Host *` ordering, negated patterns), bug fixes from open upstream pull requests, and its own releases and Homebrew tap.

---

## ✨ Features

### Server Management
- 📜 Read & display servers from your `~/.ssh/config` in a scrollable list.
- ➕ Add a new server from the UI with comprehensive SSH configuration options.
- ✏ Edit existing server entries directly from the UI with a tabbed interface.
- 🗑 Delete server entries safely.
- 📌 Pin / unpin servers to keep favorites at the top.
- 🏓 Ping server to check status.

### Quick Server Navigation
- 🔍 Fuzzy search by alias, IP, or tags.
- 🖥 One‑keypress SSH into the selected server (Enter).
- 🏷 Tag servers (e.g., prod, dev, test) for quick filtering.
- ↕️ Sort by alias or last SSH (toggle + reverse).

### On-the-fly Port Forwarding
- 🔀 Press `f` to start a local (`-L`), remote (`-R`) or dynamic/SOCKS (`-D`) forward for the selected server, without editing its config.
- 🧵 Run it as a background tunnel ("Only forward") or together with an interactive SSH session ("Forward + SSH").
- Ⓕ Servers with an active background tunnel are marked in the list. Press `x` to stop them.
- ⚠️ Background tunnels are detached and keep running after you quit lazyssh. Stop them with `x` before quitting if you don't want that.

### Advanced SSH Configuration
- 🔗 Persistent port forwarding in the config (LocalForward, RemoteForward, DynamicForward).
- 🚀 Connection multiplexing for faster subsequent connections.
- 🔐 Advanced authentication options (public key, password, agent forwarding).
- 🔒 Security settings (ciphers, MACs, key exchange algorithms).
- 🌐 Proxy settings (ProxyJump, ProxyCommand).
- ⚙️ Extensive SSH config options organized in tabbed interface.

### File Transfer
- 📁 Press `T` to copy files between your machine and the selected server with `scp`, or to open an `sftp` session.
- ⬆️ Upload a local file or folder, or ⬇️ download a remote one. Local paths autocomplete as you type (`~/` works); an empty remote path means the remote home directory.
- 📂 Folders are copied recursively. Uploading a folder turns on "Recursive" automatically.
- 💻 "Open SFTP session" suspends lazyssh and runs `sftp <alias>`; you return to lazyssh when it exits.

### Key Management
- 🔑 SSH key autocomplete with automatic detection of available keys.
- 📝 Smart key selection with support for multiple keys.
- 📤 Press `K` to install a public key on the selected server (via `ssh-copy-id`): pick one of your `~/.ssh/*.pub` keys (the server's own key is preselected) or paste someone else's.


### Upcoming
- 📁 Copy files between local and servers with an easy picker UI.
- 🔑 Generate new keypairs and deploy them.
---

## 🔐 Security Notice

lazyssh does not introduce any new security risks.
It is simply a UI/TUI wrapper around your existing `~/.ssh/config` file.

- All SSH connections are executed through your system’s native ssh binary (OpenSSH).

- Private keys, passwords, and credentials are never stored, transmitted, or modified by lazyssh.

- Your existing IdentityFile paths and ssh-agent integrations work exactly as before.

- lazyssh only reads and updates your `~/.ssh/config`. A backup of the file is created automatically before any changes.

- File permissions on your SSH config are preserved to ensure security.


## 🛡️ Config Safety: Non‑destructive writes and backups

- Non‑destructive edits: lazyssh only writes the minimal required changes to your ~/.ssh/config. It uses a parser that preserves existing comments, spacing, order, and any settings it didn’t touch. Your handcrafted comments and formatting remain intact.
- Atomic writes: updates are written to a temporary file and then atomically renamed over the original, minimizing the risk of partial writes.
- Symlinked configs: if `~/.ssh/config` is a symlink (for example into a dotfiles repo), lazyssh updates the file it points to and leaves the link in place.
- Correct host order: new hosts are added before any matching wildcard block such as a trailing `Host *`. OpenSSH uses the first value it finds for each option, so this keeps the wildcard's settings from overriding the new host's.
- Negated patterns: lines like `Host * !bastion` are written back exactly as they were, and are not listed as servers.
- Backups:
  - One‑time original backup: before lazyssh makes its first change, it creates a single snapshot named config.original.backup beside your SSH config. If this file is present, it will never be recreated or overwritten.
  - Rolling backups: on every subsequent save, lazyssh also creates a timestamped backup named like: ~/.ssh/config-<timestamp>-lazyssh.backup. The app keeps at most 10 of these backups, automatically removing the oldest ones.

## 📷 Screenshots

<div align="center">

### 🚀 Startup
<img src="./docs/loader.png" alt="App starting splash/loader" width="800" />

Clean loading screen when launching the app

---

### 📋 Server Management Dashboard
<img src="./docs/list server.png" alt="Server list view" width="900" />

Main dashboard displaying all configured servers with status indicators, pinned favorites at the top, and easy navigation

---

### 🔎 Search
<img src="./docs/search.png" alt="Fuzzy search servers" width="900" />

Fuzzy search functionality to quickly find servers by name, IP address, or tags

---

### ➕ Add/Edit Server
<img src="./docs/add server.png" alt="Add a new server" width="900" />

Tabbed interface for managing SSH connections with extensive configuration options organized into:
- **Basic** - Host, user, port, keys, tags
- **Connection** - Proxy, timeouts, multiplexing, canonicalization
- **Forwarding** - Port forwarding, X11, agent
- **Authentication** - Keys, passwords, methods, algorithm settings
- **Advanced** - Security, cryptography, environment, debugging

---

### 🔐 Connect to server
<img src="./docs/ssh.png" alt="SSH connection details" width="900" />

SSH into the selected server

</div>

---

## 📦 Installation

### Option 1: Homebrew (macOS)

```bash
brew install lord-helicon/tap/lazyssh
```

### Option 2: Download Binary from Releases

Download from [GitHub Releases](https://github.com/lord-helicon/lazyssh/releases). You can use the snippet below to automatically fetch the latest version for your OS/ARCH (Darwin/Linux and amd64/arm64 supported):

```bash
# Detect latest version
LATEST_TAG=$(curl -fsSL https://api.github.com/repos/lord-helicon/lazyssh/releases/latest | jq -r .tag_name)
# Detect your architecture (Linux reports arm64 as aarch64)
ARCH=$(uname -m); [ "$ARCH" = "aarch64" ] && ARCH=arm64
# Download the correct binary for your system
curl -LJO "https://github.com/lord-helicon/lazyssh/releases/download/${LATEST_TAG}/lazyssh_$(uname)_${ARCH}.tar.gz"
# Extract the binary
tar -xzf lazyssh_$(uname)_${ARCH}.tar.gz
# Move to /usr/local/bin or another directory in your PATH
sudo mv lazyssh /usr/local/bin/
# enjoy!
lazyssh
```

### Option 3: Build from Source

```bash
# Clone the repository
git clone https://github.com/lord-helicon/lazyssh.git
cd lazyssh

# Build for macOS
make build
./bin/lazyssh

# Or Run it directly
make run
```

---

## ⌨️ Key Bindings

| Key   | Action                        |
| ----- | ----------------------------- |
| /     | Toggle search bar             |
| ↑↓/jk | Navigate servers              |
| Enter | SSH into selected server      |
| f     | Start port forwarding         |
| x     | Stop background forwarding    |
| c     | Copy SSH command to clipboard |
| K     | Install SSH key on server     |
| T     | Transfer files / open SFTP    |
| g     | Ping selected server          |
| r     | Refresh background data       |
| a     | Add server                    |
| e     | Edit server                   |
| t     | Edit tags                     |
| d     | Delete server                 |
| p     | Pin/Unpin server              |
| s     | Toggle sort field             |
| S     | Reverse sort order            |
| q     | Quit                          |

**In Server Form:**
| Key    | Action               |
| ------ | -------------------- |
| Ctrl+H | Previous tab         |
| Ctrl+L | Next tab             |
| Ctrl+S | Save                 |
| Esc    | Cancel               |

Tip: The hint bar at the top of the list shows the most useful shortcuts.

---

## 🤝 Contributing

Contributions are welcome!

- If you spot a bug or have a feature request, please [open an issue](https://github.com/lord-helicon/lazyssh/issues).
- If you'd like to contribute, fork [lord-helicon/lazyssh](https://github.com/lord-helicon/lazyssh) and submit a pull request ❤️.

We love seeing the community make Lazyssh better 🚀

### Semantic Pull Requests

This repository enforces semantic PR titles via an automated GitHub Action. Please format your PR title as:

- type(scope): short descriptive subject
Notes:
- Scope is optional and should be one of: ui, cli, config, parser.

Allowed types in this repo:
- feat: a new feature
- fix: a bug fix
- improve: quality or UX improvements that are not a refactor or perf
- refactor: code change that neither fixes a bug nor adds a feature
- docs: documentation only changes
- test: adding or refactoring tests
- ci: CI/CD or automation changes
- chore: maintenance tasks, dependency bumps, non-code infra
- revert: reverts a previous commit

Examples:
- feat(ui): add server pinning and sorting options
- fix(parser): handle comments at end of Host blocks
- improve(cli): show friendly error when ssh binary missing
- refactor(config): simplify backup rotation logic
- docs: add installation instructions for Homebrew
- ci: cache Go toolchain and dependencies

Tip: If your PR touches multiple areas, pick the most relevant scope or omit the scope.

---

## ⭐ Support

If you find Lazyssh useful, please consider giving the repo a **star** ⭐️ and join [stargazers](https://github.com/lord-helicon/lazyssh/stargazers).

lazyssh was created by [@Adembc](https://github.com/Adembc). ☕ You can support the original author by [buying them a coffee](https://www.buymeacoffee.com/adembc) ❤️
<br/>
<a href="https://buymeacoffee.com/adembc" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/v2/default-yellow.png" width="200"></a>


---

## 🙏 Acknowledgments

- Built with [tview](https://github.com/rivo/tview) and [tcell](https://github.com/gdamore/tcell).
- Inspired by [k9s](https://github.com/derailed/k9s) and [lazydocker](https://github.com/jesseduffield/lazydocker).

