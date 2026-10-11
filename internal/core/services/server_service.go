// Copyright 2025.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package services

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/Adembc/lazyssh/internal/core/ports"
	"go.uber.org/zap"
)

type serverService struct {
	serverRepository ports.ServerRepository
	logger           *zap.SugaredLogger

	fwMu     sync.Mutex
	forwards map[string][]*os.Process

	// runInteractive runs a command attached to the terminal. Replaced in tests.
	runInteractive func(name string, args ...string) error
}

// NewServerService creates a new instance of serverService.
func NewServerService(logger *zap.SugaredLogger, sr ports.ServerRepository) ports.ServerService {
	return &serverService{
		logger:           logger,
		serverRepository: sr,
		runInteractive:   runInteractive,
	}
}

// runInteractive runs the named command with the terminal's stdin, stdout and stderr.
func runInteractive(name string, args ...string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%s not found; install OpenSSH client tools", name)
	}
	// #nosec G204 -- fixed command name; args are a key path and an alias from the user's ssh config
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ListServers returns a list of servers sorted with pinned on top.
func (s *serverService) ListServers(query string) ([]domain.Server, error) {
	servers, err := s.serverRepository.ListServers(query)
	if err != nil {
		s.logger.Errorw("failed to list servers", "error", err)
		return nil, err
	}

	// Sort: pinned first (PinnedAt non-zero), then by PinnedAt desc, then by Alias asc.
	sort.SliceStable(servers, func(i, j int) bool {
		pi := !servers[i].PinnedAt.IsZero()
		pj := !servers[j].PinnedAt.IsZero()
		if pi != pj {
			return pi
		}
		if pi && pj {
			return servers[i].PinnedAt.After(servers[j].PinnedAt)
		}
		return servers[i].Alias < servers[j].Alias
	})

	return servers, nil
}

// validateServer performs core validation of server fields.
func validateServer(srv domain.Server) error {
	if strings.TrimSpace(srv.Alias) == "" {
		return fmt.Errorf("alias is required")
	}
	if ok, _ := regexp.MatchString(`^[A-Za-z0-9_.-]+$`, srv.Alias); !ok {
		return fmt.Errorf("alias may contain letters, digits, dot, dash, underscore")
	}
	if strings.TrimSpace(srv.Host) == "" {
		return fmt.Errorf("Host/IP is required")
	}
	if ip := net.ParseIP(srv.Host); ip == nil {
		if strings.Contains(srv.Host, " ") {
			return fmt.Errorf("host must not contain spaces")
		}
		if ok, _ := regexp.MatchString(`^[A-Za-z0-9.-]+$`, srv.Host); !ok {
			return fmt.Errorf("host contains invalid characters")
		}
		if strings.HasPrefix(srv.Host, ".") || strings.HasSuffix(srv.Host, ".") {
			return fmt.Errorf("host must not start or end with a dot")
		}
		for _, lbl := range strings.Split(srv.Host, ".") {
			if lbl == "" {
				return fmt.Errorf("host must not contain empty labels")
			}
			if strings.HasPrefix(lbl, "-") || strings.HasSuffix(lbl, "-") {
				return fmt.Errorf("hostname labels must not start or end with a hyphen")
			}
		}
	}
	if srv.Port != 0 && (srv.Port < 1 || srv.Port > 65535) {
		return fmt.Errorf("port must be a number between 1 and 65535")
	}
	return nil
}

// UpdateServer updates an existing server with new details.
func (s *serverService) UpdateServer(server domain.Server, newServer domain.Server) error {
	if err := validateServer(newServer); err != nil {
		s.logger.Warnw("validation failed on update", "error", err, "server", newServer)
		return err
	}
	err := s.serverRepository.UpdateServer(server, newServer)
	if err != nil {
		s.logger.Errorw("failed to update server", "error", err, "server", server)
	}
	return err
}

// AddServer adds a new server to the repository.
func (s *serverService) AddServer(server domain.Server) error {
	if err := validateServer(server); err != nil {
		s.logger.Warnw("validation failed on add", "error", err, "server", server)
		return err
	}
	err := s.serverRepository.AddServer(server)
	if err != nil {
		s.logger.Errorw("failed to add server", "error", err, "server", server)
	}
	return err
}

// DeleteServer removes a server from the repository.
func (s *serverService) DeleteServer(server domain.Server) error {
	err := s.serverRepository.DeleteServer(server)
	if err != nil {
		s.logger.Errorw("failed to delete server", "error", err, "server", server)
	}
	return err
}

// SetPinned sets or clears a pin timestamp for the server alias.
func (s *serverService) SetPinned(alias string, pinned bool) error {
	err := s.serverRepository.SetPinned(alias, pinned)
	if err != nil {
		s.logger.Errorw("failed to set pin state", "error", err, "alias", alias, "pinned", pinned)
	}
	return err
}

// SSH starts an interactive SSH session to the given alias using the system's ssh client.
func (s *serverService) SSH(alias string) error {
	s.logger.Infow("ssh start", "alias", alias)
	cmd := exec.Command("ssh", alias)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		s.logger.Errorw("ssh command failed", "alias", alias, "error", err)
		return err
	}

	if err := s.serverRepository.RecordSSH(alias); err != nil {
		s.logger.Errorw("failed to record ssh metadata", "alias", alias, "error", err)
	}

	s.logger.Infow("ssh end", "alias", alias)
	return nil
}

// SSHWithArgs runs system ssh with provided extra args (e.g., -L/-R/-D) for the given alias.
func (s *serverService) SSHWithArgs(alias string, extraArgs []string) error {
	s.logger.Infow("ssh start (with args)", "alias", alias, "args", extraArgs)
	args := append([]string{}, extraArgs...)
	args = append(args, alias)
	// #nosec G204
	cmd := exec.Command("ssh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		s.logger.Errorw("ssh (with args) failed", "alias", alias, "error", err)
		return err
	}
	if err := s.serverRepository.RecordSSH(alias); err != nil {
		s.logger.Errorw("failed to record ssh metadata", "alias", alias, "error", err)
	}
	s.logger.Infow("ssh end (with args)", "alias", alias)
	return nil
}

// StartForward starts ssh port forwarding in the background and tracks the process.
func (s *serverService) StartForward(alias string, extraArgs []string) (int, error) {
	s.fwMu.Lock()
	if s.forwards == nil {
		s.forwards = make(map[string][]*os.Process)
	}
	s.fwMu.Unlock()

	extraArgs = append(extraArgs, "-N", alias)

	// #nosec G204
	cmd := exec.Command("ssh", extraArgs...)

	// Detach from TTY: discard stdio
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to open devnull: %w", err)
	}
	defer func() {
		if devNull != nil {
			_ = devNull.Close()
		}
	}()

	cmd.Stdin = devNull
	cmd.Stdout = devNull
	cmd.Stderr = devNull
	// Set SysProcAttr in an OS-specific way (see sysprocattr_* files)
	sysProcAttr := &syscall.SysProcAttr{}
	setDetach(sysProcAttr)
	cmd.SysProcAttr = sysProcAttr

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start ssh: %w", err)
	}

	proc := cmd.Process
	if proc == nil {
		return 0, fmt.Errorf("process is nil after start")
	}
	pid := proc.Pid

	// Track process
	s.fwMu.Lock()
	s.forwards[alias] = append(s.forwards[alias], proc)
	s.fwMu.Unlock()

	// Cleanup on exit
	go func(a string, c *exec.Cmd, dn *os.File) {
		_ = c.Wait()
		_ = dn.Close()

		s.fwMu.Lock()
		defer s.fwMu.Unlock()

		procs := s.forwards[a]
		if len(procs) == 0 {
			return
		}

		filtered := make([]*os.Process, 0, len(procs))
		for _, p := range procs {
			if p != nil && p.Pid != pid {
				filtered = append(filtered, p)
			}
		}

		if len(filtered) == 0 {
			delete(s.forwards, a)
		} else {
			s.forwards[a] = filtered
		}
	}(alias, cmd, devNull)

	devNull = nil // Prevent defer from closing it

	return pid, nil
}

// StopForwarding kills all active forward processes for the alias.
func (s *serverService) StopForwarding(alias string) error {
	s.fwMu.Lock()
	procs := s.forwards[alias]
	delete(s.forwards, alias)
	s.fwMu.Unlock()

	if len(procs) == 0 {
		return nil
	}

	var errs []error
	for _, p := range procs {
		if p != nil {
			if err := p.Signal(syscall.SIGTERM); err != nil {
				// If SIGTERM fails, try SIGKILL
				if killErr := p.Kill(); killErr != nil {
					errs = append(errs, fmt.Errorf("failed to kill pid %d: %w", p.Pid, killErr))
				}
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors stopping forwards: %v", errs)
	}
	return nil
}

// IsForwarding reports whether there is at least one active forward for alias.
func (s *serverService) IsForwarding(alias string) bool {
	s.fwMu.Lock()
	defer s.fwMu.Unlock()
	return len(s.forwards[alias]) > 0
}

// Ping checks if the server is reachable on its SSH port.
func (s *serverService) Ping(server domain.Server) (bool, time.Duration, error) {
	start := time.Now()

	host, port, ok := resolveSSHDestination(server.Alias)
	if !ok {

		host = strings.TrimSpace(server.Host)
		if host == "" {
			host = server.Alias
		}
		if server.Port > 0 {
			port = server.Port
		} else {
			port = 22
		}
	}
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	dialer := net.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return false, time.Since(start), err
	}
	_ = conn.Close()
	return true, time.Since(start), nil
}

// resolveSSHDestination uses `ssh -G <alias>` to extract HostName and Port from the user's SSH config.
// Returns host, port, ok where ok=false if resolution failed.
func resolveSSHDestination(alias string) (string, int, bool) {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return "", 0, false
	}
	cmd := exec.Command("ssh", "-G", alias)
	out, err := cmd.Output()
	if err != nil {
		return "", 0, false
	}
	host := ""
	port := 0
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "hostname ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				host = parts[1]
			}
		}
		if strings.HasPrefix(line, "port ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if p, err := strconv.Atoi(parts[1]); err == nil {
					port = p
				}
			}
		}
	}
	if host == "" {
		host = alias
	}
	if port == 0 {
		port = 22
	}
	return host, port, true
}

// InstallPublicKey appends the public key at pubKeyPath to the server's
// authorized_keys using ssh-copy-id.
func (s *serverService) InstallPublicKey(alias, pubKeyPath string) error {
	s.logger.Infow("ssh-copy-id start", "alias", alias, "key", pubKeyPath)
	if err := s.runInteractive("ssh-copy-id", sshCopyIDArgs(alias, pubKeyPath)...); err != nil {
		s.logger.Errorw("ssh-copy-id failed", "alias", alias, "key", pubKeyPath, "error", err)
		return err
	}
	s.logger.Infow("ssh-copy-id end", "alias", alias)
	return nil
}

// InstallPublicKeyContent installs a pasted public key line. ssh-copy-id only
// reads keys from files, so the key is written to a private temp file first.
func (s *serverService) InstallPublicKeyContent(alias, key string) error {
	f, err := os.CreateTemp("", "lazyssh-*.pub")
	if err != nil {
		return fmt.Errorf("create temp key file: %w", err)
	}
	path := f.Name()
	defer func() {
		if rmErr := os.Remove(path); rmErr != nil {
			s.logger.Warnw("failed to remove temp key file", "path", path, "error", rmErr)
		}
	}()

	_, writeErr := f.WriteString(strings.TrimSpace(key) + "\n")
	if closeErr := f.Close(); writeErr == nil {
		writeErr = closeErr
	}
	if writeErr != nil {
		return fmt.Errorf("write temp key file: %w", writeErr)
	}
	return s.InstallPublicKey(alias, path)
}

// sshCopyIDArgs builds the ssh-copy-id arguments for installing pubKeyPath.
// Without -f, ssh-copy-id requires the matching private key next to the .pub
// file, so keys without one (e.g. a colleague's key) are installed with -f,
// which also skips the check for keys already on the server.
func sshCopyIDArgs(alias, pubKeyPath string) []string {
	var args []string
	if _, err := os.Stat(strings.TrimSuffix(pubKeyPath, ".pub")); err != nil {
		args = append(args, "-f")
	}
	return append(args, "-i", pubKeyPath, alias)
}

// UploadFiles copies localPath to alias:remotePath with scp. An empty
// remotePath means the remote home directory.
func (s *serverService) UploadFiles(alias, localPath, remotePath string, recursive bool) error {
	return s.runTransfer("upload", alias, "scp", scpUploadArgs(alias, localPath, remotePath, recursive))
}

// DownloadFiles copies alias:remotePath to localPath with scp.
func (s *serverService) DownloadFiles(alias, remotePath, localPath string, recursive bool) error {
	return s.runTransfer("download", alias, "scp", scpDownloadArgs(alias, remotePath, localPath, recursive))
}

// OpenSFTP starts an interactive sftp session with alias.
func (s *serverService) OpenSFTP(alias string) error {
	return s.runTransfer("sftp", alias, "sftp", sftpArgs(alias))
}

func (s *serverService) runTransfer(kind, alias, name string, args []string) error {
	s.logger.Infow(kind+" start", "alias", alias, "command", name, "args", args)
	if err := s.runInteractive(name, args...); err != nil {
		s.logger.Errorw(kind+" failed", "alias", alias, "error", err)
		return err
	}
	s.logger.Infow(kind+" end", "alias", alias)
	return nil
}

func scpUploadArgs(alias, localPath, remotePath string, recursive bool) []string {
	return scpArgs(recursive, safeLocalPath(localPath), alias+":"+remotePath)
}

func scpDownloadArgs(alias, remotePath, localPath string, recursive bool) []string {
	return scpArgs(recursive, alias+":"+remotePath, safeLocalPath(localPath))
}

func scpArgs(recursive bool, src, dst string) []string {
	var args []string
	if recursive {
		args = append(args, "-r")
	}
	return append(args, src, dst)
}

func sftpArgs(alias string) []string {
	return []string{alias}
}

// safeLocalPath keeps scp from reading a local path that starts with "-" as an option.
func safeLocalPath(p string) string {
	if strings.HasPrefix(p, "-") {
		return "./" + p
	}
	return p
}

// GenerateKey creates a new key pair at path with ssh-keygen. ssh-keygen asks
// for the passphrase itself on the terminal; lazyssh never sees the key.
func (s *serverService) GenerateKey(path, keyType, comment string) error {
	args, err := sshKeygenArgs(path, keyType, comment)
	if err != nil {
		return err
	}
	s.logger.Infow("ssh-keygen start", "path", path, "type", keyType)
	if err := s.runInteractive("ssh-keygen", args...); err != nil {
		s.logger.Errorw("ssh-keygen failed", "path", path, "type", keyType, "error", err)
		return err
	}
	s.logger.Infow("ssh-keygen end", "path", path)
	return nil
}

// sshKeygenArgs builds the ssh-keygen arguments for the given key type.
func sshKeygenArgs(path, keyType, comment string) ([]string, error) {
	var args []string
	switch keyType {
	case "ed25519", "ecdsa":
		args = []string{"-t", keyType}
	case "rsa":
		args = []string{"-t", "rsa", "-b", "4096"}
	default:
		return nil, fmt.Errorf("unsupported key type %q", keyType)
	}
	return append(args, "-f", path, "-C", comment), nil
}
