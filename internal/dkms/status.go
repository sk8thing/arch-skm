package dkms

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/sk8thing/arch-skm/internal/command"
)

const (
	defaultDir     = "/etc/dkms"
	kernelSignConf = "kernel-sign.conf"
)

type Module struct {
	Name       string
	Configured bool
	Dangling   bool
}

type statusEntry struct {
	Name    string
	Version string
	Kernel  string
}

func (e statusEntry) Spec() string {
	if e.Version == "" {
		return e.Name
	}

	return e.Name + "/" + e.Version
}

func (m Module) State() string {
	if m.Dangling {
		return "dangling"
	}
	if m.Configured {
		return "configured"
	}

	return "installed"
}

func Status(ctx context.Context) ([]Module, error) {
	entries, err := installedEntries(ctx)
	if err != nil {
		return nil, err
	}

	installed := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		installed[entry.Name] = struct{}{}
	}

	modules := make([]Module, 0, len(entries))
	for _, name := range statusNames(entries) {
		configured, err := isConfigured(moduleConf(defaultDir, name))
		if err != nil {
			return nil, err
		}

		modules = append(modules, Module{Name: name, Configured: configured})
	}

	dangling, err := danglingConfigs(defaultDir, installed)
	if err != nil {
		return nil, err
	}
	for _, name := range dangling {
		modules = append(modules, Module{Name: name, Configured: true, Dangling: true})
	}

	return modules, nil
}

func danglingConfigs(dir string, installed map[string]struct{}) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}

	var names []string
	for _, entry := range entries {
		if entry.Name() == kernelSignConf {
			continue
		}

		name, ok := strings.CutSuffix(entry.Name(), ".conf")
		if !ok || name == "" {
			continue
		}
		if _, ok := installed[name]; ok {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("access %s: %w", filepath.Join(dir, entry.Name()), err)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			continue
		}

		names = append(names, name)
	}

	return names, nil
}

func installedEntries(ctx context.Context) ([]statusEntry, error) {
	output, err := statusOutput(ctx)
	if err != nil {
		return nil, err
	}

	return parseStatusEntries(output), nil
}

func isConfigured(path string) (bool, error) {
	if _, err := os.Lstat(path); err == nil {
		return true, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return false, fmt.Errorf("access %s: %w", path, err)
	}

	return false, nil
}

func moduleConf(dir, name string) string {
	return filepath.Join(dir, name+".conf")
}

func statusNames(entries []statusEntry) []string {
	var names []string
	seen := make(map[string]bool)

	for _, entry := range entries {
		if seen[entry.Name] {
			continue
		}

		seen[entry.Name] = true
		names = append(names, entry.Name)
	}

	return names
}

func parseStatusEntries(output string) []statusEntry {
	var entries []statusEntry

	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		idx := strings.LastIndexByte(line, ':')
		if idx < 0 {
			continue
		}
		state := strings.Fields(line[idx+1:])
		if len(state) == 0 || !strings.EqualFold(state[0], "installed") {
			continue
		}

		fields := strings.Split(line[:idx], ",")
		name, version, _ := strings.Cut(strings.TrimSpace(fields[0]), "/")
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		entry := statusEntry{
			Name:    name,
			Version: strings.TrimSpace(version),
		}
		if len(fields) > 1 {
			entry.Kernel = strings.TrimSpace(fields[1])
		}

		entries = append(entries, entry)
	}

	return entries
}

func statusOutput(ctx context.Context) (string, error) {
	return command.Runner{}.Output(ctx, command.Options{}, "dkms", "status")
}
