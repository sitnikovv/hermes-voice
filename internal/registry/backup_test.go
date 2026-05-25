package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBackupFileCopiesExactBytesToDefaultDirectory(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "registry.yaml")
	content := []byte("schema_version: 1\n# exact bytes\n")
	if err := os.WriteFile(source, content, 0o600); err != nil {
		t.Fatal(err)
	}

	backupPath, err := BackupFile(source, "", time.Date(2026, 5, 25, 8, 40, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("BackupFile() error = %v", err)
	}
	if filepath.Dir(backupPath) != filepath.Join(dir, ".registry-backups") {
		t.Fatalf("backup dir = %q, want default .registry-backups", filepath.Dir(backupPath))
	}
	if !strings.Contains(filepath.Base(backupPath), "registry-20260525-084000.yaml") {
		t.Fatalf("backup base = %q, want timestamped registry yaml", filepath.Base(backupPath))
	}
	got, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Fatalf("backup content = %q, want exact %q", got, content)
	}
}

func TestBackupFileDoesNotCreateBackupDirectoryWhenSourceMissing(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.yaml")
	backupDir := filepath.Join(dir, "backups")

	if _, err := BackupFile(missing, backupDir, time.Date(2026, 5, 25, 8, 42, 0, 0, time.UTC)); err == nil {
		t.Fatal("BackupFile() error = nil, want missing source error")
	}
	if _, err := os.Stat(backupDir); !os.IsNotExist(err) {
		t.Fatalf("backup dir stat error = %v, want not exist", err)
	}
}

func TestBackupFileUsesRequestedDirectoryAndAvoidsOverwrite(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "registry.yaml")
	if err := os.WriteFile(source, []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	backupDir := filepath.Join(dir, "backups")
	now := time.Date(2026, 5, 25, 8, 41, 0, 0, time.UTC)

	first, err := BackupFile(source, backupDir, now)
	if err != nil {
		t.Fatalf("first BackupFile() error = %v", err)
	}
	if err := os.WriteFile(source, []byte("two"), 0o600); err != nil {
		t.Fatal(err)
	}
	second, err := BackupFile(source, backupDir, now)
	if err != nil {
		t.Fatalf("second BackupFile() error = %v", err)
	}
	if first == second {
		t.Fatalf("backup paths equal %q, want collision-safe unique path", first)
	}
	firstBytes, _ := os.ReadFile(first)
	secondBytes, _ := os.ReadFile(second)
	if string(firstBytes) != "one" || string(secondBytes) != "two" {
		t.Fatalf("backup contents first=%q second=%q", firstBytes, secondBytes)
	}
}
