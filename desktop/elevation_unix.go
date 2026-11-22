//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// FindDaemonBinary locates netportal-daemon in system paths or next to the desktop binary.
func FindDaemonBinary() (string, error) {
	if exe, err := os.Executable(); err == nil {
		local := filepath.Join(filepath.Dir(exe), "netportal-daemon")
		if _, err := os.Stat(local); err == nil {
			_ = os.Chmod(local, 0o755)
			return local, nil
		}
	}
	for _, sysPath := range []string{"/usr/lib/netportal/netportal-daemon", "/usr/local/bin/netportal-daemon"} {
		if _, err := os.Stat(sysPath); err == nil {
			return sysPath, nil
		}
	}
	return "", fmt.Errorf("netportal-daemon binary not found")
}

// LaunchElevatedDaemon spawns a single persistent pkexec session for portable Linux runs.
func LaunchElevatedDaemon(daemonPath string) error {
	_ = os.Chmod(daemonPath, 0o755)
	cmd := exec.Command("pkexec", daemonPath)
	return cmd.Start()
}
