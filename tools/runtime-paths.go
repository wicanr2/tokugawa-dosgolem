package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
)

var buildVersion = "development"
var buildCommit = ""
var playMode bool
var runtimeDiagnostics = true
var runtimeAssetAuditPath string

type runtimePaths struct{ Project, Original, Output string }

var hostPaths = runtimePaths{"/project", "/original", "/output"}
var runtimeInputs = struct {
	sync.Mutex
	Paths map[string]bool
}{Paths: map[string]bool{}}

func (p runtimePaths) resolve(name string) string {
	portable := filepath.ToSlash(name)
	// Temporary files return concrete paths. Keep them stable when a caller
	// passes one through another file helper, including nested test mounts.
	for _, target := range []string{p.Project, p.Original, p.Output} {
		root := strings.TrimSuffix(filepath.ToSlash(target), "/")
		if portable == root || strings.HasPrefix(portable, root+"/") {
			return name
		}
	}
	for _, root := range []struct{ Prefix, Target string }{{"/project", p.Project}, {"/original", p.Original}, {"/output", p.Output}} {
		if portable == root.Prefix {
			return root.Target
		}
		if strings.HasPrefix(portable, root.Prefix+"/") {
			return filepath.Join(root.Target, filepath.FromSlash(strings.TrimPrefix(portable, root.Prefix+"/")))
		}
	}
	return name
}

func resolveRuntimePath(name string) string { return hostPaths.resolve(name) }

func defaultStateDirectory() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "windows":
		for _, name := range []string{"LOCALAPPDATA", "APPDATA"} {
			if value := os.Getenv(name); filepath.IsAbs(value) {
				return filepath.Join(value, "tokugawa-dosgolem"), nil
			}
		}
		return filepath.Join(homeDir, "AppData", "Local", "tokugawa-dosgolem"), nil
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", "tokugawa-dosgolem"), nil
	default:
		if value := os.Getenv("XDG_DATA_HOME"); filepath.IsAbs(value) {
			return filepath.Join(value, "tokugawa-dosgolem"), nil
		}
		return filepath.Join(homeDir, ".local", "share", "tokugawa-dosgolem"), nil
	}
}

func configureRuntimePaths(resourceDir, gameDir, stateDir string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	base := filepath.Dir(exe)
	if filepath.Base(base) == "MacOS" && filepath.Base(filepath.Dir(base)) == "Contents" {
		base = filepath.Join(filepath.Dir(base), "Resources")
	}
	paths := hostPaths
	if playMode {
		paths.Project, paths.Original = filepath.Join(base, "runtime"), filepath.Join(base, "game")
		paths.Output, err = defaultStateDirectory()
		if err != nil {
			return err
		}
	}
	for _, pair := range []struct {
		Value  string
		Target *string
	}{{resourceDir, &paths.Project}, {gameDir, &paths.Original}, {stateDir, &paths.Output}} {
		if pair.Value != "" {
			*pair.Target = pair.Value
		}
		*pair.Target, err = filepath.Abs(*pair.Target)
		if err != nil {
			return err
		}
	}
	for _, name := range []string{paths.Project, paths.Original} {
		info, e := os.Stat(name)
		if e != nil || !info.IsDir() {
			return fmt.Errorf("找不到資源或原版資料夾：%s", name)
		}
	}
	original, err := filepath.EvalSymlinks(paths.Original)
	if err != nil {
		return err
	}
	state := paths.Output
	if actual, e := filepath.EvalSymlinks(state); e == nil {
		state = actual
	}
	rel, err := filepath.Rel(original, state)
	if err == nil && (rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))) {
		return fmt.Errorf("存檔資料夾必須與原版資料分開")
	}
	for _, name := range []string{paths.Output, filepath.Join(paths.Output, "startup-probe"), filepath.Join(paths.Output, "screenshots")} {
		if err := os.MkdirAll(name, 0700); err != nil {
			return err
		}
	}
	hostPaths = paths
	return nil
}

func recordRuntimeInput(name string) {
	if runtimeAssetAuditPath == "" {
		return
	}
	runtimeInputs.Lock()
	defer runtimeInputs.Unlock()
	if len(runtimeInputs.Paths) < 4096 {
		runtimeInputs.Paths[name] = true
	}
}

func saveRuntimeInputAudit() {
	if runtimeAssetAuditPath == "" {
		return
	}
	runtimeInputs.Lock()
	paths := make([]string, 0, len(runtimeInputs.Paths))
	for p := range runtimeInputs.Paths {
		paths = append(paths, p)
	}
	runtimeInputs.Unlock()
	sort.Strings(paths)
	data, err := json.MarshalIndent(map[string]any{"version": buildVersion, "paths": paths, "resource_root": hostPaths.Project, "original_root": hostPaths.Original}, "", "  ")
	if err != nil {
		panic(err)
	}
	if err = os.WriteFile(resolveRuntimePath(runtimeAssetAuditPath), data, 0600); err != nil {
		panic(err)
	}
}

func readRuntimeFile(name string) ([]byte, error) {
	name = resolveRuntimePath(name)
	data, err := os.ReadFile(name)
	if err == nil {
		recordRuntimeInput(name)
	}
	return data, err
}
func openRuntimeFile(name string) (*os.File, error) {
	name = resolveRuntimePath(name)
	f, err := os.Open(name)
	if err == nil {
		recordRuntimeInput(name)
	}
	return f, err
}
func statRuntimeFile(name string) (os.FileInfo, error) { return os.Stat(resolveRuntimePath(name)) }
func readRuntimeDirectory(name string) ([]os.DirEntry, error) {
	return os.ReadDir(resolveRuntimePath(name))
}
func createRuntimeFile(name string) (*os.File, error) { return os.Create(resolveRuntimePath(name)) }
func createRuntimeTemp(dir, pattern string) (*os.File, error) {
	return os.CreateTemp(resolveRuntimePath(dir), pattern)
}
func makeRuntimeTemp(dir, pattern string) (string, error) {
	return os.MkdirTemp(resolveRuntimePath(dir), pattern)
}
func writeRuntimeFile(name string, b []byte, mode fs.FileMode) error {
	return os.WriteFile(resolveRuntimePath(name), b, mode)
}
func renameRuntimeFile(from, to string) error {
	return os.Rename(resolveRuntimePath(from), resolveRuntimePath(to))
}
func makeRuntimeDirectory(name string, mode fs.FileMode) error {
	return os.Mkdir(resolveRuntimePath(name), mode)
}
func makeRuntimeDirectories(name string, mode fs.FileMode) error {
	return os.MkdirAll(resolveRuntimePath(name), mode)
}

func runtimeWorkingData(program string) (string, error) {
	if !playMode {
		return makeRuntimeTemp("/output/startup-probe", "state-")
	}
	path := resolveRuntimePath("/output/saved-data/" + program)
	return path, os.MkdirAll(path, 0700)
}
