package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/dosgolem/internal/machine"
)

func TestRuntimeRootMapping(t *testing.T) {
	base := filepath.Join(t.TempDir(), "中文 與空格")
	p := runtimePaths{filepath.Join(base, "runtime"), filepath.Join(base, "game"), filepath.Join(base, "state")}
	for _, row := range []struct{ In, Want string }{
		{"/project/assets/art.png", filepath.Join(p.Project, "assets", "art.png")},
		{filepath.Join("/project", "assets", "art.png"), filepath.Join(p.Project, "assets", "art.png")},
		{"/original/DX5.DAT", filepath.Join(p.Original, "DX5.DAT")},
		{"/output/startup-probe/status.json", filepath.Join(p.Output, "startup-probe", "status.json")},
		{"/project", p.Project}, {"/project-other/file", "/project-other/file"}, {"relative.png", "relative.png"},
	} {
		if got := p.resolve(row.In); got != row.Want {
			t.Fatalf("%q: %q != %q", row.In, got, row.Want)
		}
	}
}

func TestRuntimeResolvedPathsAreStable(t *testing.T) {
	p := runtimePaths{filepath.FromSlash("/project/resources"), filepath.FromSlash("/original/game"), filepath.FromSlash("/output/user-data")}
	for _, name := range []string{"/project/art.png", "/original/DX5.DAT", "/output/startup-probe/audio.pcm"} {
		resolved := p.resolve(name)
		if again := p.resolve(resolved); again != resolved {
			t.Fatalf("path resolved twice: %q -> %q -> %q", name, resolved, again)
		}
	}
}

func TestOriginalOverlaySurvivesReopening(t *testing.T) {
	root := t.TempDir()
	base, state := filepath.Join(root, "唯讀 原版"), filepath.Join(root, "使用者 存檔")
	for _, p := range []string{base, state} {
		if err := os.Mkdir(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(base, "DX5.DAT"), []byte("original-data"), 0600); err != nil {
		t.Fatal(err)
	}
	files, err := machine.OpenDirectoryOverlayFiles(base, state)
	if err != nil {
		t.Fatal(err)
	}
	f, err := files.OpenWrite("dx5.dat", true)
	if err != nil {
		t.Fatal(err)
	}
	writer, ok := f.(io.Writer)
	if !ok {
		t.Fatal("OpenWrite did not provide a writer")
	}
	if _, err = writer.Write([]byte("saved")); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	if err = files.Close(); err != nil {
		t.Fatal(err)
	}
	files, err = machine.OpenDirectoryOverlayFiles(base, state)
	if err != nil {
		t.Fatal(err)
	}
	defer files.Close()
	f, err = files.OpenRead("DX5.DAT")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	got, err := io.ReadAll(f)
	if err != nil || string(got) != "savednal-data" {
		t.Fatalf("persistent data: %q %v", got, err)
	}
	untouched, err := os.ReadFile(filepath.Join(base, "DX5.DAT"))
	if err != nil || string(untouched) != "original-data" {
		t.Fatal("original data changed", err)
	}
	if unsafe, err := machine.OpenDirectoryOverlayFiles(base, base); err == nil {
		unsafe.Close()
		t.Fatal("shared source/state accepted")
	}
}

func TestRuntimeStateCannotWriteInsideOriginals(t *testing.T) {
	root := t.TempDir()
	original := filepath.Join(root, "original")
	if err := os.Mkdir(original, 0700); err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{original, filepath.Join(original, "saved-data")} {
		if err := configureRuntimePaths(root, original, state); err == nil {
			t.Fatal("original/state overlap accepted")
		}
	}
	if _, err := os.Stat(filepath.Join(original, "saved-data")); !os.IsNotExist(err) {
		t.Fatal("created an output inside originals", err)
	}
}
