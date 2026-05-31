package capyhost

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReadConfigExtensionRouting asserts that ReadConfig transpiles capy
// source for `.capy` and `.wave` extensions but passes other formats
// through to the YAML/JSON parser unchanged.
//
// Regression context: until this matrix landed, only `.capy` triggered
// the capy frontend, so a user with `server.wave` got their file fed
// into the YAML parser raw — which choked on the first backtick block
// or capy-specific syntax (e.g. mapping values are not allowed in
// flow-style JSON-like response templates).
func TestReadConfigExtensionRouting(t *testing.T) {
	// Minimal capy source. The YAML parser cannot read this (no colons,
	// backtick scalars), so if ReadConfig returns it verbatim instead
	// of transpiling, downstream yaml.Unmarshal will reject it.
	capySrc := "default\n    port 9803\n"

	// Minimal YAML source. The capy parser would reject it (`:` is not
	// a capy token), so if ReadConfig transpiles by accident, this case
	// would fail.
	yamlSrc := "default:\n  port: 9803\n"

	cases := []struct {
		name    string
		ext     string
		src     string
		wantSub string // a substring expected in the YAML output
		wantErr bool
	}{
		{name: "capy lowercase", ext: ".capy", src: capySrc, wantSub: "port: 9803"},
		{name: "capy uppercase", ext: ".CAPY", src: capySrc, wantSub: "port: 9803"},
		{name: "wave lowercase", ext: ".wave", src: capySrc, wantSub: "port: 9803"},
		{name: "wave uppercase", ext: ".WAVE", src: capySrc, wantSub: "port: 9803"},
		{name: "wave mixed case", ext: ".Wave", src: capySrc, wantSub: "port: 9803"},
		{name: "yaml passthrough", ext: ".yaml", src: yamlSrc, wantSub: "port: 9803"},
		{name: "yml passthrough", ext: ".yml", src: yamlSrc, wantSub: "port: 9803"},
		{name: "json passthrough", ext: ".json", src: `{"default":{"port":9803}}`, wantSub: `"port":9803`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "server"+tc.ext)
			if err := os.WriteFile(path, []byte(tc.src), 0o644); err != nil {
				t.Fatal(err)
			}
			out, err := ReadConfig(path)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tc.wantErr)
			}
			if !strings.Contains(string(out), tc.wantSub) {
				t.Errorf("output %q missing %q", string(out), tc.wantSub)
			}
		})
	}
}

// TestReadConfigWaveFileWithRichCapySyntax loads a non-trivial capy
// file (multi-storage + route block + backtick template) via .wave
// extension. The regression this catches: backtick blocks inside flow-
// style-looking content (`{"id": ...}`) parse fine under capy but blow
// up under YAML.
func TestReadConfigWaveFileWithRichCapySyntax(t *testing.T) {
	capySrc := `default
    port 9803
storage
    notes
        type sqlite
        path ./notes.db
routes
    -
        path /api/notes
        method POST
        type storage-access
        storage-access
            source notes
            execute ` + "`INSERT INTO notes (note) VALUES ({{value \"note\"}})`" + `
            output_template ` + "`{\"id\": {{.LastInsertID}}, \"success\": true}`" + `
`
	dir := t.TempDir()
	path := filepath.Join(dir, "server.wave")
	if err := os.WriteFile(path, []byte(capySrc), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := ReadConfig(path)
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}
	// The transpiled YAML must contain the storage source and the
	// preserved backtick template.
	for _, want := range []string{"port: 9803", "source: notes", "INSERT INTO notes", `"id":`} {
		if !strings.Contains(string(out), want) {
			t.Errorf("transpiled output missing %q\n--- output ---\n%s", want, string(out))
		}
	}
}

// TestReadConfigMissingFile preserves the os.ReadFile error surface so
// callers can still distinguish "no such file" from a parse failure.
func TestReadConfigMissingFile(t *testing.T) {
	_, err := ReadConfig(filepath.Join(t.TempDir(), "nope.wave"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "no such file") && !strings.Contains(err.Error(), "cannot find") {
		t.Errorf("unexpected error shape: %v", err)
	}
}

// TestIsCapyExt is a small surface check on the extension matcher so a
// future contributor adding a third alias has an obvious failing test
// to update.
func TestIsCapyExt(t *testing.T) {
	cases := map[string]bool{
		".capy": true,
		".CAPY": true,
		".wave": true,
		".WAVE": true,
		".Wave": true,
		".yaml": false,
		".yml":  false,
		".json": false,
		".txt":  false,
		"":      false,
		".cap":  false, // typo guard — must not silently match
	}
	for ext, want := range cases {
		if got := isCapyExt(ext); got != want {
			t.Errorf("isCapyExt(%q) = %v, want %v", ext, got, want)
		}
	}
}
