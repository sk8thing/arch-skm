package patch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const pkgbuildFixture = `pkgname=('linux'
         'linux-headers'
         "$pkgbase-docs"
)

prepare() {
	local builddir="${srcdir}"
	msg "Preparing"
}

build() {
	make htmldocs &
	local pid_docs=$!
	wait "${pid_docs}"
}

_package-headers() {
	local builddir="${pkgdir}/usr/lib/modules/${_kernver}/build"
	msg "Header build"
}
`

func writePKGBUILD(t *testing.T, content string, configAtStart bool) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "PKGBUILD"), []byte(content), 0644); err != nil {
		t.Fatalf("write PKGBUILD: %v", err)
	}
	if configAtStart {
		if err := os.WriteFile(filepath.Join(dir, "config.x86_64"), []byte("# config\n"), 0644); err != nil {
			t.Fatalf("write kernel config: %v", err)
		}
	}

	return dir
}

func readPKGBUILD(t *testing.T, dir string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(dir, "PKGBUILD"))
	if err != nil {
		t.Fatalf("read PKGBUILD: %v", err)
	}

	return string(data)
}

func TestPatchPKGBUILDConfigAtStart(t *testing.T) {
	dir := writePKGBUILD(t, pkgbuildFixture, true)

	if err := PatchPKGBUILD(dir); err != nil {
		t.Fatalf("PatchPKGBUILD() error: %v", err)
	}

	content := readPKGBUILD(t, dir)

	if !strings.Contains(content, "prepare() {\n  # "+SigningMarker) {
		t.Error("prepare() prefix not inserted at start")
	}
	if !strings.Contains(content, `./skm gen-keys '../config*'`) {
		t.Error("prepare() should read ../config* when configs exist at start")
	}
	if strings.Contains(content, `'../.config'`) {
		t.Error("prepare() should not read ../.config when configs exist at start")
	}

	for _, want := range []string{"# make htmldocs", "# local pid_docs=$!", `# wait "`} {
		if !strings.Contains(content, want) {
			t.Errorf("build() should contain commented line %q", want)
		}
	}
	if strings.Contains(content, "\nmake htmldocs") {
		t.Error("build() still contains an uncommented make htmldocs")
	}

	if !strings.Contains(content, `certs_local_dst="${builddir}/certs-local"`) {
		t.Error("package-headers suffix should reference the existing builddir variable")
	}
	if !strings.Contains(content, `certs_local_src="../certs-local"`) {
		t.Error("package-headers should use ../certs-local when configs exist at start")
	}
	if !strings.Contains(content, `${certs_local_src}/skm install-certs $certs_local_dst`) {
		t.Error("package-headers suffix not appended")
	}

	if !strings.Contains(content, `  # "$pkgbase-docs"`) {
		t.Error("pkgname entries should comment out the -docs package")
	}
	if strings.Contains(content, "\n         \"$pkgbase-docs\"\n") {
		t.Error("pkgname still contains an uncommented -docs entry")
	}
}

func TestPatchPKGBUILDConfigGeneratedInPrepare(t *testing.T) {
	dir := writePKGBUILD(t, pkgbuildFixture, false)

	if err := PatchPKGBUILD(dir); err != nil {
		t.Fatalf("PatchPKGBUILD() error: %v", err)
	}

	content := readPKGBUILD(t, dir)

	if strings.Contains(content, "this is added at the start") {
		t.Error("prepare() should not get the prefix when config is generated in prepare()")
	}
	if !strings.Contains(content, "this is added at the end of prepare()") {
		t.Error("prepare() suffix not appended")
	}
	if !strings.Contains(content, `./skm gen-keys '../.config'`) {
		t.Error("prepare() should read ../.config when config is generated in prepare()")
	}

	if found := strings.Contains(content, `msg "Preparing"`); !found {
		t.Error("original prepare() body lost")
	}

	if !strings.Contains(content, `certs_local_src="./certs-local"`) {
		t.Error("package-headers should use ./certs-local when config is generated in prepare()")
	}
}

func TestPatchPKGBUILDAltBuilddirVariable(t *testing.T) {
	content := strings.Replace(pkgbuildFixture,
		`local builddir="${pkgdir}/usr/lib/modules/${_kernver}/build"`,
		`local _builddir="${pkgdir}/usr/lib/modules/${_kernver}/build"`, 1)
	dir := writePKGBUILD(t, content, true)

	if err := PatchPKGBUILD(dir); err != nil {
		t.Fatalf("PatchPKGBUILD() error: %v", err)
	}

	got := readPKGBUILD(t, dir)
	if !strings.Contains(got, `certs_local_dst="${_builddir}/certs-local"`) {
		t.Error("package-headers suffix should adopt the _builddir variable name")
	}
}

func TestPatchPKGBUILDIsIdempotent(t *testing.T) {
	dir := writePKGBUILD(t, pkgbuildFixture, true)

	if err := PatchPKGBUILD(dir); err != nil {
		t.Fatalf("first PatchPKGBUILD() error: %v", err)
	}
	first := readPKGBUILD(t, dir)

	if err := PatchPKGBUILD(dir); err != nil {
		t.Fatalf("second PatchPKGBUILD() error: %v", err)
	}
	second := readPKGBUILD(t, dir)

	if strings.Count(second, SigningMarker) != strings.Count(first, SigningMarker) {
		t.Error("patching an already patched PKGBUILD duplicated the signing blocks")
	}
	if second != first {
		t.Error("patching an already patched PKGBUILD changed its content")
	}
}

func TestPatchPKGBUILDMissingPKGBUILD(t *testing.T) {
	if err := PatchPKGBUILD(t.TempDir()); err == nil {
		t.Fatal("PatchPKGBUILD() succeeded, want error")
	}
}

func TestPatchPKGBUILDMissingFunction(t *testing.T) {
	content := strings.Replace(pkgbuildFixture, "\nbuild() {\n\tmake htmldocs &\n\tlocal pid_docs=$!\n\twait \"${pid_docs}\"\n}\n", "\n", 1)
	dir := writePKGBUILD(t, content, false)

	err := PatchPKGBUILD(dir)
	if err == nil {
		t.Fatal("PatchPKGBUILD() succeeded, want error")
	}
	if !strings.Contains(err.Error(), "build function block") {
		t.Errorf("PatchPKGBUILD() error = %q, want it to mention the build function block", err)
	}
}

func TestHasKernelConfig(t *testing.T) {
	dir := t.TempDir()
	if has, err := hasKernelConfig(dir); has || err != nil {
		t.Errorf("hasKernelConfig(empty) = %v, %v; want false, nil", has, err)
	}

	if err := os.Mkdir(filepath.Join(dir, "config"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if has, err := hasKernelConfig(dir); has || err != nil {
		t.Errorf("hasKernelConfig(dir only) = %v, %v; want false, nil", has, err)
	}

	if err := os.WriteFile(filepath.Join(dir, "config.x86_64"), []byte("# config\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if has, err := hasKernelConfig(dir); !has || err != nil {
		t.Errorf("hasKernelConfig(with file) = %v, %v; want true, nil", has, err)
	}
}
