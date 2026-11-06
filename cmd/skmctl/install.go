package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sk8thing/arch-skm/internal/command"
	"github.com/sk8thing/arch-skm/internal/kernel"
	"github.com/sk8thing/arch-skm/internal/pacman"
	"github.com/spf13/cobra"
)

const installExample = `  # Install linux-hardened kernel
  skmctl install linux-hardened`

var installCmd = &cobra.Command{
	Use:       "install [kernel]",
	Short:     "Install built kernel packages using pacman",
	Example:   installExample,
	Args:      cobra.ExactArgs(1),
	ValidArgs: kernel.Supported(),
	RunE: func(cmd *cobra.Command, args []string) error {
		t, err := resolveTarget(args[0])
		if err != nil {
			return err
		}

		if err := validateBuildDir(t.Dir); err != nil {
			return err
		}

		packages, err := listBuiltPackages(cmd, t.Dir, t.Variant)
		if err != nil {
			return err
		}

		noConfirm, _ := cmd.Flags().GetBool("noconfirm")
		fmt.Fprintf(cmd.OutOrStdout(), "Installing %s\n", strings.Join(packages, " "))

		if err := runPacman(cmd, packages, noConfirm); err != nil {
			return err
		}

		ensureKernelIgnored(cmd, t.Variant, noConfirm)

		return nil
	},
}

func init() {
	installCmd.Flags().Bool("noconfirm", false, "Do not ask for confirmation (--noconfirm)")

	rootCmd.AddCommand(installCmd)
}

func listBuiltPackages(cmd *cobra.Command, dir string, variant string) ([]string, error) {
	output, err := command.Runner{}.Output(cmd.Context(), command.Options{Dir: dir}, "makepkg", "--packagelist")
	if err != nil {
		return nil, fmt.Errorf("makepkg --packagelist: %w", err)
	}

	var packages []string
	for line := range strings.SplitSeq(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !filepath.IsAbs(line) {
			line = filepath.Join(dir, line)
		}

		info, err := os.Stat(line)
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("built package %q not found; run 'skmctl build %s' first", line, variant)
		}
		if err != nil {
			return nil, fmt.Errorf("access package %q: %w", line, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("package path %q is not a regular file", line)
		}

		packages = append(packages, line)
	}

	if len(packages) == 0 {
		return nil, fmt.Errorf("makepkg --packagelist returned no packages for %s", variant)
	}

	return packages, nil
}

func runPacman(cmd *cobra.Command, packages []string, noConfirm bool) error {
	args := []string{"-U"}
	if noConfirm {
		args = append(args, "--noconfirm")
	}
	args = append(args,
		"--overwrite", "/etc/dkms/kernel-sign.conf",
		"--overwrite", "/etc/dkms/kernel-sign.sh",
	)
	args = append(args, packages...)

	runner := command.Runner{
		Stdin:  cmd.InOrStdin(),
		Stdout: cmd.OutOrStdout(),
		Stderr: cmd.ErrOrStderr(),
	}

	if err := runner.Run(cmd.Context(), command.Options{Sudo: true}, "pacman", args...); err != nil {
		return err
	}

	return nil
}

func ensureKernelIgnored(cmd *cobra.Command, variant string, noConfirm bool) {
	names := []string{variant, variant + "-headers"}

	ignored, err := pacman.Ignored(cmd.Context())
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: read pacman IgnorePkg: %v\n", err)
		return
	}

	missing := pacman.Missing(ignored, names)
	if len(missing) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "%s already ignored by pacman\n", strings.Join(names, " and "))
		return
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "%s not ignored by pacman; a system upgrade could replace the locally signed kernel\n", strings.Join(missing, " and "))

	if !noConfirm && isInteractive(cmd.InOrStdin()) {
		answer, err := promptYesNo(cmd.InOrStdin(), out, fmt.Sprintf("Add %s to IgnorePkg in %s?", strings.Join(missing, " and "), pacman.ConfigPath))
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: read confirmation: %v\n", err)
			return
		}
		if !answer {
			fmt.Fprintf(out, "Add \"IgnorePkg = %s\" to %s manually to keep the local kernel\n", strings.Join(names, " "), pacman.ConfigPath)
			return
		}
	}

	content, err := os.ReadFile(pacman.ConfigPath)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: %v\n", err)
		return
	}

	updated, changed := pacman.AddIgnore(string(content), missing)
	if !changed {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not update IgnorePkg in %s; add \"IgnorePkg = %s\" manually\n", pacman.ConfigPath, strings.Join(names, " "))
		return
	}

	if err := pacman.WriteConfig(cmd.Context(), updated, cmd.ErrOrStderr()); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: %v\n", err)
		return
	}

	fmt.Fprintf(out, "%s added to IgnorePkg in %s\n", strings.Join(missing, " and "), pacman.ConfigPath)
}

func isInteractive(stdin io.Reader) bool {
	file, ok := stdin.(*os.File)
	if !ok {
		return false
	}

	info, err := file.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}

func promptYesNo(stdin io.Reader, stdout io.Writer, question string) (bool, error) {
	reader := bufio.NewReader(stdin)
	for {
		fmt.Fprintf(stdout, "%s [Y/n] ", question)

		line, err := reader.ReadString('\n')
		if err != nil && line == "" {
			return false, err
		}

		switch strings.ToLower(strings.TrimSpace(line)) {
		case "", "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			fmt.Fprintln(stdout, "Please answer 'y' or 'n'")
		}
	}
}
