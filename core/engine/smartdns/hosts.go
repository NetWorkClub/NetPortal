// Package smartdns implements atomic hosts file patching (BOM-free) and a local
// HTTP CONNECT proxy for VPN-less Smart DNS / Smart Hosts workflows.
package smartdns

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const (
	hostsBeginMarker = "# BEGIN NETPORTAL SMARTDNS"
	hostsEndMarker   = "# END NETPORTAL SMARTDNS"
)

// DefaultHostsPath returns the OS hosts file location.
func DefaultHostsPath() string {
	if runtime.GOOS == "windows" {
		sysRoot := os.Getenv("SystemRoot")
		if sysRoot == "" {
			sysRoot = `C:\Windows`
		}
		return filepath.Join(sysRoot, "System32", "drivers", "etc", "hosts")
	}
	return "/etc/hosts"
}

// ApplyHostsEntries atomically writes a NetPortal-managed block into hostsPath
// in strict UTF-8 without BOM, preserving all unrelated user entries.
func ApplyHostsEntries(hostsPath string, entries map[string]string) error {
	if hostsPath == "" {
		hostsPath = DefaultHostsPath()
	}
	raw, err := os.ReadFile(hostsPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	// Strip UTF-8 BOM if present on existing hosts file.
	raw = bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})

	cleaned := stripManagedBlock(string(raw))
	if len(entries) == 0 {
		return writeAtomicBOMFree(hostsPath, cleaned)
	}

	domains := make([]string, 0, len(entries))
	for d := range entries {
		domains = append(domains, d)
	}
	sort.Strings(domains)

	var block strings.Builder
	block.WriteString(hostsBeginMarker + "\n")
	for _, d := range domains {
		ip := strings.TrimSpace(entries[d])
		if ip == "" || strings.TrimSpace(d) == "" {
			continue
		}
		block.WriteString(fmt.Sprintf("%s %s\n", ip, strings.TrimSpace(d)))
	}
	block.WriteString(hostsEndMarker + "\n")

	var out strings.Builder
	out.WriteString(cleaned)
	if cleaned != "" && !strings.HasSuffix(cleaned, "\n") {
		out.WriteByte('\n')
	}
	out.WriteString(block.String())

	return writeAtomicBOMFree(hostsPath, out.String())
}

// RemoveHostsEntries removes any NetPortal-managed block from hostsPath.
func RemoveHostsEntries(hostsPath string) error {
	return ApplyHostsEntries(hostsPath, nil)
}

func stripManagedBlock(content string) string {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	out := make([]string, 0, len(lines))
	inBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == hostsBeginMarker {
			inBlock = true
			continue
		}
		if trimmed == hostsEndMarker {
			inBlock = false
			continue
		}
		if !inBlock {
			out = append(out, line)
		}
	}
	res := strings.Join(out, "\n")
	return strings.TrimRight(res, "\n") + "\n"
}

func writeAtomicBOMFree(path, content string) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".netportal-hosts-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write([]byte(content)); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}
