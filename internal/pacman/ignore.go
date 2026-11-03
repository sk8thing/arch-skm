// Package pacman reads and edits the pacman configuration.
package pacman

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
)

const ConfigPath = "/etc/pacman.conf"

var ignorePkgLine = regexp.MustCompile(`(?i)^(\s*IgnorePkg\s*=\s*)(.*)$`)

func Ignored(ctx context.Context) ([]string, error) {
	if path, err := exec.LookPath("pacman-conf"); err == nil {
		if output, err := exec.CommandContext(ctx, path, "IgnorePkg").Output(); err == nil {
			return strings.Fields(string(output)), nil
		}
	}

	content, err := os.ReadFile(ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", ConfigPath, err)
	}

	return parseIgnored(string(content)), nil
}

func Missing(ignored, wanted []string) []string {
	present := make(map[string]struct{}, len(ignored))
	for _, name := range ignored {
		present[name] = struct{}{}
	}

	var missing []string
	for _, name := range wanted {
		if _, ok := present[name]; !ok {
			missing = append(missing, name)
		}
	}

	return missing
}

func AddIgnore(content string, names []string) (string, bool) {
	if len(names) == 0 {
		return content, false
	}

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		match := ignorePkgLine.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		value, comment := splitComment(match[2])
		fields := strings.Fields(value)
		if containsAll(fields, names) {
			return content, false
		}
		for _, name := range names {
			if !slices.Contains(fields, name) {
				fields = append(fields, name)
			}
		}

		updated := strings.TrimRight(match[1], " \t") + " " + strings.Join(fields, " ")
		if comment != "" {
			updated += " " + comment
		}
		lines[i] = updated

		return strings.Join(lines, "\n"), true
	}

	insert, ok := optionsSectionEnd(lines)
	if !ok {
		return content, false
	}

	updated := make([]string, 0, len(lines)+1)
	updated = append(updated, lines[:insert]...)
	updated = append(updated, "IgnorePkg = "+strings.Join(names, " "))
	updated = append(updated, lines[insert:]...)

	return strings.Join(updated, "\n"), true
}

func WriteConfig(ctx context.Context, content string, stderr io.Writer) error {
	info, err := os.Stat(ConfigPath)
	if err != nil {
		return fmt.Errorf("access %s: %w", ConfigPath, err)
	}

	if os.Geteuid() == 0 {
		if err := os.WriteFile(ConfigPath, []byte(content), info.Mode().Perm()); err != nil {
			return fmt.Errorf("write %s: %w", ConfigPath, err)
		}
		return nil
	}

	sudoPath, err := exec.LookPath("sudo")
	if err != nil {
		return fmt.Errorf("find sudo: %w", err)
	}

	cmd := exec.CommandContext(ctx, sudoPath, "tee", ConfigPath)
	cmd.Stdin = strings.NewReader(content)
	cmd.Stdout = io.Discard
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s tee %s: %w", sudoPath, ConfigPath, err)
	}

	return nil
}

func parseIgnored(content string) []string {
	var ignored []string
	for line := range strings.SplitSeq(content, "\n") {
		match := ignorePkgLine.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		value, _ := splitComment(match[2])
		ignored = append(ignored, strings.Fields(value)...)
	}

	return ignored
}

func splitComment(value string) (string, string) {
	if idx := strings.IndexByte(value, '#'); idx >= 0 {
		return strings.TrimSpace(value[:idx]), strings.TrimSpace(value[idx:])
	}

	return strings.TrimSpace(value), ""
}

func containsAll(values []string, wanted []string) bool {
	for _, want := range wanted {
		if !slices.Contains(values, want) {
			return false
		}
	}

	return true
}

func optionsSectionEnd(lines []string) (int, bool) {
	start := -1
	for i, line := range lines {
		if strings.EqualFold(strings.TrimSpace(line), "[options]") {
			start = i
			break
		}
	}
	if start == -1 {
		return 0, false
	}

	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "[") {
			end = i
			break
		}
	}

	for end > start+1 && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}

	return end, true
}
