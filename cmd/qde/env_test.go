package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnv(t *testing.T) {
	p := filepath.Join(t.TempDir(), ".env")
	_ = os.WriteFile(p, []byte("# comment\n\nQDE_A=1\nQDE_B = \"two\"\nQDE_SET=file\nnot a pair\n"), 0o600)
	t.Setenv("QDE_SET", "env")
	t.Setenv("QDE_A", "")
	_ = os.Unsetenv("QDE_A")
	loadDotEnv(p)
	if os.Getenv("QDE_A") != "1" || os.Getenv("QDE_B") != "two" {
		t.Errorf("A=%q B=%q", os.Getenv("QDE_A"), os.Getenv("QDE_B"))
	}
	if os.Getenv("QDE_SET") != "env" {
		t.Error("existing env var was overridden")
	}
	t.Cleanup(func() { _ = os.Unsetenv("QDE_B") })
	loadDotEnv(filepath.Join(t.TempDir(), "missing")) // must not panic
}
