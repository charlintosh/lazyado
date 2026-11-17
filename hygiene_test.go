package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	emailPattern     = regexp.MustCompile(`(?i)[a-z0-9._+-]+(@|%40)([a-z0-9-]+\.)+[a-z]{2,}`)
	allowedEmail     = regexp.MustCompile(`(?i)(@|%40)([a-z0-9-]+\.)*(example\.(com|org|net)|example)$`)
	adoPATPattern    = regexp.MustCompile(`\b[a-z2-7]{52}\b|\b[A-Za-z0-9]{84}\b`)
	basicAuthPattern = regexp.MustCompile(`Basic [A-Za-z0-9+/]{20,}={0,2}`)
	patValuePattern  = regexp.MustCompile(`(?im)^\s*pat:\s*["']?([^"'\s#]+)`)
	patPlaceholders  = map[string]bool{"your-personal-access-token": true}
	skippedDirs      = map[string]bool{".git": true, "dist": true, "node_modules": true}
	skippedFiles     = map[string]bool{"go.sum": true}
	scannedExts      = map[string]bool{".go": true, ".md": true, ".yaml": true, ".yml": true, ".json": true, ".sh": true, ".ps1": true, ".txt": true, ".env": true}
)

func TestRepositoryHasNoHardcodedUserData(t *testing.T) {
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skippedDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || skippedFiles[d.Name()] || !scannedExts[filepath.Ext(path)] || isUntracked(path) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		checkContent(t, path, string(data))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func checkContent(t *testing.T, path, content string) {
	t.Helper()
	for _, email := range emailPattern.FindAllString(content, -1) {
		if !allowedEmail.MatchString(email) && !strings.HasPrefix(path, ".agents") && !strings.HasPrefix(path, ".github"+string(filepath.Separator)+"skills") {
			t.Errorf("%s: email %q is not on a reserved example domain; use name@example.com in fixtures and docs", path, email)
		}
	}
	if m := adoPATPattern.FindString(content); m != "" {
		t.Errorf("%s: contains a string shaped like an Azure DevOps PAT", path)
	}
	if m := basicAuthPattern.FindString(content); m != "" {
		t.Errorf("%s: contains an encoded Basic auth credential", path)
	}
	for _, m := range patValuePattern.FindAllStringSubmatch(content, -1) {
		if !patPlaceholders[m[1]] {
			t.Errorf("%s: sets pat to a non-placeholder value", path)
		}
	}
}

func isUntracked(path string) bool {
	return path == "config.yaml" || path == ".env"
}
