package sign

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/sk8thing/arch-skm/internal/utils"
)

const (
	keysDirName     = "keys"
	signingKeyFile  = "signing_key.pem"
	signingCertFile = "signing_crt.crt"
	khashFile       = "khash"
	defaultHashType = "sha512"
)

func SignModules(ctx context.Context, paths []string, jobs int) error {
	if len(paths) == 0 {
		return fmt.Errorf("no modules to sign")
	}

	signFile, keyFile, certFile, hashType, err := signingFiles()
	if err != nil {
		return err
	}

	return signAll(ctx, paths, jobs, func(ctx context.Context, path string) error {
		return signModule(ctx, signFile, keyFile, certFile, hashType, path)
	})
}

func signAll(ctx context.Context, paths []string, jobs int, sign func(context.Context, string) error) error {
	paths = uniquePaths(paths)
	workers := resolveJobs(jobs, len(paths))
	errs := make([]error, len(paths))

	type job struct {
		index int
		path  string
	}

	work := make(chan job)
	var wg sync.WaitGroup

	for range workers {
		wg.Go(func() {
			for item := range work {
				if err := ctx.Err(); err != nil {
					errs[item.index] = fmt.Errorf("sign module %q: %w", item.path, err)
					continue
				}
				if err := sign(ctx, item.path); err != nil {
					errs[item.index] = fmt.Errorf("sign module %q: %w", item.path, err)
				}
			}
		})
	}

dispatch:
	for index, path := range paths {
		if err := ctx.Err(); err != nil {
			for i := index; i < len(paths); i++ {
				errs[i] = fmt.Errorf("sign module %q: %w", paths[i], err)
			}
			break
		}

		select {
		case work <- job{index: index, path: path}:
		case <-ctx.Done():
			for i := index; i < len(paths); i++ {
				errs[i] = fmt.Errorf("sign module %q: %w", paths[i], ctx.Err())
			}
			break dispatch
		}
	}
	close(work)
	wg.Wait()

	return errors.Join(errs...)
}

func uniquePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	unique := make([]string, 0, len(paths))

	for _, path := range paths {
		key := path
		if absPath, err := filepath.Abs(path); err == nil {
			key = absPath
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, path)
	}

	return unique
}

func resolveJobs(jobs, modules int) int {
	if jobs <= 0 {
		jobs = runtime.GOMAXPROCS(0)
	}
	if jobs > modules {
		jobs = modules
	}
	if jobs < 1 {
		jobs = 1
	}

	return jobs
}

func signingFiles() (signFile string, keyFile string, certFile string, hashType string, err error) {
	exePath, err := utils.GetExecutablePath()
	if err != nil {
		return "", "", "", "", fmt.Errorf("get executable path: %w", err)
	}

	certDir := filepath.Dir(exePath)
	buildDir := filepath.Dir(certDir)
	keysDir := filepath.Join(certDir, keysDirName)

	signFile = filepath.Join(buildDir, "scripts", "sign-file")
	keyFile = filepath.Join(keysDir, signingKeyFile)
	certFile = filepath.Join(keysDir, signingCertFile)

	files := []struct {
		name string
		path string
	}{
		{"kernel signing tool", signFile},
		{"private key", keyFile},
		{"certificate", certFile},
	}

	for _, file := range files {
		if _, err := os.Stat(file.path); err != nil {
			return "", "", "", "", fmt.Errorf("%s %q: %w", file.name, file.path, err)
		}
	}

	hashType, err = readKhash(filepath.Join(keysDir, khashFile))
	if err != nil {
		return "", "", "", "", err
	}

	return signFile, keyFile, certFile, hashType, nil
}

func readKhash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return defaultHashType, nil
	}
	if err != nil {
		return "", fmt.Errorf("read kernel hash %q: %w", path, err)
	}

	hashType := strings.TrimSpace(string(data))
	if hashType == "" {
		return defaultHashType, nil
	}

	return hashType, nil
}
