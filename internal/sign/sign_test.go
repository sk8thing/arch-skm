package sign

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestCompressRoundtrip(t *testing.T) {
	data := bytes.Repeat([]byte("kernel module payload\x00\x01\x02\n"), 500)

	for _, ext := range []string{".zst", ".xz", ".gz"} {
		t.Run(ext, func(t *testing.T) {
			compressed, err := compress(ext, data)
			if err != nil {
				t.Fatalf("compress(%q) error: %v", ext, err)
			}

			decompressed, err := decompress(ext, compressed)
			if err != nil {
				t.Fatalf("decompress(%q) error: %v", ext, err)
			}

			if !bytes.Equal(decompressed, data) {
				t.Errorf("roundtrip for %q failed: %d -> %d bytes, content mismatch", ext, len(compressed), len(decompressed))
			}
		})
	}
}

func TestDecompressUnsupported(t *testing.T) {
	if _, err := decompress(".txt", []byte("data")); err == nil {
		t.Error("decompress(.txt) succeeded, want error")
	}
	if _, err := compress(".txt", []byte("data")); err == nil {
		t.Error("compress(.txt) succeeded, want error")
	}
}

func TestDecompressPlainModule(t *testing.T) {
	data := []byte("plain module")
	got, err := decompress(".ko", data)
	if err != nil {
		t.Fatalf("decompress(.ko) error: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("decompress(.ko) = %q, want %q", got, data)
	}
}

func TestDecompressInvalidData(t *testing.T) {
	for _, ext := range []string{".zst", ".xz", ".gz"} {
		if _, err := decompress(ext, []byte("not a compressed module")); err == nil {
			t.Errorf("decompress(%q) accepted invalid data", ext)
		}
	}
}

func TestIsCompressed(t *testing.T) {
	for _, ext := range []string{".zst", ".xz", ".gz"} {
		if !isCompressed(ext) {
			t.Errorf("isCompressed(%q) = false, want true", ext)
		}
	}
	for _, ext := range []string{".ko", ".txt", ""} {
		if isCompressed(ext) {
			t.Errorf("isCompressed(%q) = true, want false", ext)
		}
	}
}

func TestSignatureTail(t *testing.T) {
	sizes := []int{50, signatureChunk, signatureChunk + 1, signatureChunk * 2, signatureChunk * 100}

	for _, size := range sizes {
		data := bytes.Repeat([]byte("x"), size)
		data[len(data)-1] = 'y' // mark the very end

		tail := signatureTail(data)

		if want := min(size, signatureChunk); len(tail) != want {
			t.Errorf("signatureTail(len %d) returned %d bytes, want %d", size, len(tail), want)
		}
		if !bytes.HasSuffix(data, tail) {
			t.Error("signatureTail is not the end of the data")
		}
		if easy := signatureTail(nil); easy != nil {
			t.Error("signatureTail(nil) is not nil")
		}
	}
}

func TestIsSigned(t *testing.T) {
	noise := bytes.Repeat([]byte("\x00\x01\x02\x03"), 200) // 800 bytes

	signed := append([]byte{}, noise...)
	signed = append(signed, []byte("\n"+signatureMarker+"\n")...)
	if !isSigned(signed) {
		t.Error("isSigned(signed) = false, want true")
	}

	if isSigned(noise) {
		t.Error("isSigned(unsigned) = true, want false")
	}

	// A marker beyond the signature tail is not detected sign-file style.
	far := make([]byte, signatureChunk*3)
	copy(far, []byte(signatureMarker))
	if isSigned(far) {
		t.Error("isSigned(marker in front but far from tail) = true, want false")
	}
}

func TestIsSignedFile(t *testing.T) {
	dir := t.TempDir()

	t.Run("signed", func(t *testing.T) {
		path := filepath.Join(dir, "signed.ko")
		data := append(bytes.Repeat([]byte("module data "), 50), []byte(signatureMarker)...)
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatalf("write module: %v", err)
		}

		if got, err := isSignedFile(path); err != nil || got != true {
			t.Errorf("isSignedFile(signed) = %v, %v; want true, nil", got, err)
		}
	})

	t.Run("unsigned", func(t *testing.T) {
		path := filepath.Join(dir, "unsigned.ko")
		if err := os.WriteFile(path, bytes.Repeat([]byte("module data "), 50), 0644); err != nil {
			t.Fatalf("write module: %v", err)
		}

		if got, err := isSignedFile(path); err != nil || got != false {
			t.Errorf("isSignedFile(unsigned) = %v, %v; want false, nil", got, err)
		}
	})

	t.Run("small signed", func(t *testing.T) {
		path := filepath.Join(dir, "small.ko")
		if err := os.WriteFile(path, []byte("tiny "+signatureMarker), 0644); err != nil {
			t.Fatalf("write module: %v", err)
		}

		if got, err := isSignedFile(path); err != nil || got != true {
			t.Errorf("isSignedFile(small signed) = %v, %v; want true, nil", got, err)
		}
	})

	t.Run("missing", func(t *testing.T) {
		if _, err := isSignedFile(filepath.Join(dir, "nonexistent.ko")); err == nil {
			t.Error("isSignedFile(missing) succeeded, want error")
		}
	})
}

func TestWriteAndCopyTempModule(t *testing.T) {
	dir := t.TempDir()

	srcPath := filepath.Join(dir, "src.ko")
	srcData := []byte("module contents")
	if err := os.WriteFile(srcPath, srcData, 0644); err != nil {
		t.Fatalf("write source module: %v", err)
	}

	copied, err := copyTempModule(dir, srcPath)
	if err != nil {
		t.Fatalf("copyTempModule() error: %v", err)
	}
	defer os.Remove(copied)

	got, err := os.ReadFile(copied)
	if err != nil {
		t.Fatalf("read copied module: %v", err)
	}
	if !bytes.Equal(got, srcData) {
		t.Errorf("copied content = %q, want %q", got, srcData)
	}
	if filepath.Dir(copied) != dir || !strings.HasPrefix(filepath.Base(copied), ".skm-") {
		t.Errorf("copied module path = %q, want a .skm-* temp name inside %q", copied, dir)
	}

	written, err := writeTempModule(dir, srcData)
	if err != nil {
		t.Fatalf("writeTempModule() error: %v", err)
	}
	defer os.Remove(written)

	got, err = os.ReadFile(written)
	if err != nil {
		t.Fatalf("read written module: %v", err)
	}
	if !bytes.Equal(got, srcData) {
		t.Errorf("written content = %q, want %q", got, srcData)
	}
}

func TestReplaceModule(t *testing.T) {
	dir := t.TempDir()

	finalPath := filepath.Join(dir, "module.ko")
	if err := os.WriteFile(finalPath, []byte("old"), 0644); err != nil {
		t.Fatalf("write module: %v", err)
	}

	tmpPath := filepath.Join(dir, "tmp.ko")
	if err := os.WriteFile(tmpPath, []byte("new contents"), 0600); err != nil {
		t.Fatalf("write tmp module: %v", err)
	}

	if err := replaceModule(tmpPath, finalPath, 0755); err != nil {
		t.Fatalf("replaceModule() error: %v", err)
	}

	got, err := os.ReadFile(finalPath)
	if err != nil {
		t.Fatalf("read module: %v", err)
	}
	if string(got) != "new contents" {
		t.Errorf("module content = %q, want %q", got, "new contents")
	}
	if info, err := os.Stat(finalPath); err != nil || info.Mode().Perm() != 0755 {
		t.Errorf("module permissions = %v, %v; want 0755, nil", info, err)
	}
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Errorf("tmp module still exists: %v", err)
	}
}

func TestFindModules(t *testing.T) {
	dir := t.TempDir()

	for name, content := range map[string]string{
		"a.ko":     "plain",
		"b.ko.zst": "zst",
		"d.ko.gz":  "gz",
		"m4.ko":    "plain up2date",
		"e.text":   "not a module",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "subdir.a.ko"), 0755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}

	modules, err := FindModules(dir)
	if err != nil {
		t.Fatalf("FindModules() error: %v", err)
	}

	want := []string{
		filepath.Join(dir, "a.ko"),
		filepath.Join(dir, "b.ko.zst"),
		filepath.Join(dir, "d.ko.gz"),
		filepath.Join(dir, "m4.ko"),
	}
	if len(modules) != len(want) {
		t.Fatalf("FindModules() = %v, want %v", modules, want)
	}
	for i, module := range modules {
		if module != want[i] {
			t.Errorf("FindModules()[%d] = %q, want %q", i, module, want[i])
		}
	}
}

func TestFindModulesErrors(t *testing.T) {
	if _, err := FindModules(filepath.Join(t.TempDir(), "nonexistent")); err == nil {
		t.Error("FindModules(nonexistent) succeeded, want error")
	}

	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("data"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := FindModules(file); err == nil {
		t.Error("FindModules(file) succeeded, want error")
	}
}

func TestUniquePaths(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	a := filepath.Join(dir, "a.ko")
	b := filepath.Join(dir, "b.ko")

	got := uniquePaths([]string{
		a,
		"./a.ko", // same file as a via the current directory
		b,
		"b.ko", // same file as b via the current directory
		filepath.Join(dir, "c.ko"),
	})

	want := []string{a, b, filepath.Join(dir, "c.ko")}
	if len(got) != len(want) {
		t.Fatalf("uniquePaths() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("uniquePaths()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestResolveJobs(t *testing.T) {
	tests := []struct {
		name    string
		jobs    int
		modules int
		exact   int
	}{
		{"explicit", 3, 10, 3},
		{"capped by modules", 7, 3, 3},
		{"one worker enough", 1, 5, 1},
		{"zero jobs with no modules", 0, 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveJobs(tt.jobs, tt.modules); got != tt.exact {
				t.Errorf("resolveJobs(%d, %d) = %d, want %d", tt.jobs, tt.modules, got, tt.exact)
			}
		})
	}

	// Zero/negative job counts fall back to the number of CPUs.
	if got := resolveJobs(-1, 100); got != runtime.GOMAXPROCS(0) {
		t.Errorf("resolveJobs(-1, 100) = %d, want %d", got, runtime.GOMAXPROCS(0))
	}
}

func TestSignAll(t *testing.T) {
	paths := []string{"m1.ko", "m2.ko", "m3.ko"}

	var mu sync.Mutex
	var signed []string

	sign := func(_ context.Context, path string) error {
		mu.Lock()
		defer mu.Unlock()

		signed = append(signed, path)

		return nil
	}

	if err := signAll(context.Background(), paths, 3, sign); err != nil {
		t.Fatalf("signAll() error: %v", err)
	}

	counts := make(map[string]int)
	for _, path := range signed {
		counts[path]++
	}
	for _, path := range paths {
		if counts[path] != 1 {
			t.Errorf("path %q signed %d times, want once", path, counts[path])
		}
	}
}

func TestSignAllDuplicates(t *testing.T) {
	var mu sync.Mutex
	counts := make(map[string]int)

	sign := func(_ context.Context, path string) error {
		mu.Lock()
		defer mu.Unlock()

		counts[path]++

		return nil
	}

	paths := []string{"m1.ko", "m1.ko", "m2.ko", "m1.ko"}
	if err := signAll(context.Background(), paths, 0, sign); err != nil {
		t.Fatalf("signAll() error: %v", err)
	}

	if counts["m1.ko"] != 1 {
		t.Errorf("m1.ko signed %d times, want once", counts["m1.ko"])
	}
}

func TestSignAllErrors(t *testing.T) {
	paths := []string{"m1.ko", "m2.ko", "m3.ko"}

	sign := func(_ context.Context, path string) error {
		if path == "m2.ko" {
			return fmt.Errorf("boom")
		}

		return nil
	}

	err := signAll(context.Background(), paths, 2, sign)
	if err == nil {
		t.Fatal("signAll() succeeded, want error")
	}
	if !strings.Contains(err.Error(), "m2.ko") {
		t.Errorf("signAll() error = %q, want it to mention the failed module", err)
	}
}

func TestSignAllContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	sign := func(_ context.Context, _ string) error {
		t.Error("signAll signed modules despite a cancelled context")

		return nil
	}

	if err := signAll(ctx, []string{"m1.ko", "m2.ko"}, 2, sign); err == nil {
		t.Fatal("signAll() succeeded with a cancelled context, want error")
	}
}

func TestSignAllContextCancelledMidway(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	var mu sync.Mutex
	var signed []string

	sign := func(_ context.Context, path string) error {
		cancel() // Cancel after completing a module.

		mu.Lock()
		defer mu.Unlock()

		signed = append(signed, path)

		return nil
	}

	if err := signAll(ctx, []string{"m1.ko", "m2.ko", "m3.ko"}, 1, sign); err == nil {
		t.Fatal("signAll() succeeded, want error after cancellation")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(signed) != 1 {
		t.Errorf("signed %d modules before cancellation took effect, want 1", len(signed))
	}
}

func TestResolveJobsSanity(t *testing.T) {
	// The job count must never exceed the module count or default worker
	// count and never drop below one.
	for _, jobs := range []int{-100, -1, 0, 1, 2, 1000} {
		for _, modules := range []int{0, 1, 3} {
			if got := resolveJobs(jobs, modules); got < 1 || got > max(modules, 1) {
				t.Errorf("resolveJobs(%d, %d) = %d, out of bounds", jobs, modules, got)
			}
		}
	}
}

func TestReadKhash(t *testing.T) {
	dir := t.TempDir()

	got, err := readKhash(filepath.Join(dir, "missing"))
	if err != nil || got != defaultHashType {
		t.Errorf("readKhash(missing) = %q, %v; want %q, nil", got, err, defaultHashType)
	}

	path := filepath.Join(dir, "khash")
	if err := os.WriteFile(path, []byte("sha256\n"), 0644); err != nil {
		t.Fatalf("write khash: %v", err)
	}
	if got, err := readKhash(path); err != nil || got != "sha256" {
		t.Errorf("readKhash() = %q, %v; want sha256, nil", got, err)
	}

	if err := os.WriteFile(path, []byte("\n"), 0644); err != nil {
		t.Fatalf("write khash: %v", err)
	}
	if got, err := readKhash(path); err != nil || got != defaultHashType {
		t.Errorf("readKhash(empty) = %q, %v; want %q, nil", got, err, defaultHashType)
	}
}
