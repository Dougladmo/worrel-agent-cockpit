package workspace

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initBareSource cria um repo git de origem (não-bare, com 1 commit) e devolve
// um URL file:// clonável, para testar CloneRepo offline.
func initBareSource(t *testing.T) string {
	t.Helper()
	src := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = src
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("init")
	if err := os.WriteFile(filepath.Join(src, "README.md"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "init")
	return "file://" + src
}

func TestCloneRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git ausente")
	}
	url := initBareSource(t)
	root := t.TempDir()
	m := New(root)

	dest, err := m.CloneRepo("proj", url)
	if err != nil {
		t.Fatalf("CloneRepo: %v", err)
	}
	want := filepath.Join(root, "repos", "proj")
	if dest != want {
		t.Fatalf("dest = %s, want %s", dest, want)
	}
	if _, err := os.Stat(filepath.Join(dest, "README.md")); err != nil {
		t.Fatalf("arquivo clonado ausente: %v", err)
	}
}

func TestCloneRepoEmptyURL(t *testing.T) {
	m := New(t.TempDir())
	if _, err := m.CloneRepo("p", ""); err == nil {
		t.Fatal("esperava erro para url vazia")
	}
}

func TestCloneRepoDestExists(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git ausente")
	}
	url := initBareSource(t)
	root := t.TempDir()
	m := New(root)
	if _, err := m.CloneRepo("dup", url); err != nil {
		t.Fatalf("primeiro clone: %v", err)
	}
	if _, err := m.CloneRepo("dup", url); err == nil {
		t.Fatal("esperava erro para destino existente")
	}
}
