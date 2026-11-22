//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

// FindDaemonBinary locates netportal-daemon.exe next to the desktop executable.
func FindDaemonBinary() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	candidate := filepath.Join(filepath.Dir(exe), "netportal-daemon.exe")
	if _, err := os.Stat(candidate); err == nil {
		return candidate, nil
	}
	return "", fmt.Errorf("netportal-daemon.exe not found next to %s", exe)
}

// LaunchElevatedDaemon triggers a single UAC prompt via ShellExecuteW("runas")
// for portable Windows runs when NetPortalService is not installed.
func LaunchElevatedDaemon(daemonPath string) error {
	shell32 := syscall.NewLazyDLL("shell32.dll")
	proc := shell32.NewProc("ShellExecuteW")

	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString(daemonPath)
	dir, _ := syscall.UTF16PtrFromString(filepath.Dir(daemonPath))

	ret, _, err := proc.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		0,
		uintptr(unsafe.Pointer(dir)),
		0, // SW_HIDE
	)
	if ret <= 32 {
		return fmt.Errorf("ShellExecuteW runas failed (code %d): %w", ret, err)
	}
	return nil
}
