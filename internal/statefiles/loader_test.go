package statefiles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStateFilePathsFollowsSymlinkDirectories(t *testing.T) {
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "repo")
	stateDir := filepath.Join(tempDir, "states")
	linkedDir := filepath.Join(stateDir, "terraform-repo")

	if err := os.MkdirAll(filepath.Join(repoDir, "ecs", "beijing", "devops"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "ecs", "beijing", "devops", "terraform.tfstate"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "ecs", "beijing", "devops", "terraform.tfstate.backup"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(repoDir, linkedDir); err != nil {
		t.Fatal(err)
	}

	paths, err := stateFilePaths(stateDir)
	if err != nil {
		t.Fatalf("stateFilePaths() error = %v", err)
	}
	if len(paths) != 1 {
		t.Fatalf("paths = %#v, want one .tfstate file", paths)
	}
	if filepath.Base(paths[0]) != "terraform.tfstate" {
		t.Fatalf("path = %q, want terraform.tfstate", paths[0])
	}
}
