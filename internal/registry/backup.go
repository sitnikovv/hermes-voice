package registry

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BackupFile copies sourcePath byte-for-byte into backupDir and returns the backup path.
// If backupDir is empty, <source-dir>/.registry-backups is used.
func BackupFile(sourcePath, backupDir string, now time.Time) (string, error) {
	if strings.TrimSpace(sourcePath) == "" {
		return "", fmt.Errorf("registry backup: source path is required")
	}
	if now.IsZero() {
		now = time.Now()
	}
	if backupDir == "" {
		backupDir = filepath.Join(filepath.Dir(sourcePath), ".registry-backups")
	}

	source, err := os.Open(sourcePath)
	if err != nil {
		return "", err
	}
	defer source.Close()

	if err := os.MkdirAll(backupDir, 0o700); err != nil {
		return "", err
	}

	base := strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))
	ext := filepath.Ext(sourcePath)
	if ext == "" {
		ext = ".bak"
	}
	stamp := now.Format("20060102-150405")
	for i := 0; ; i++ {
		name := fmt.Sprintf("%s-%s%s", base, stamp, ext)
		if i > 0 {
			name = fmt.Sprintf("%s-%s-%02d%s", base, stamp, i, ext)
		}
		backupPath := filepath.Join(backupDir, name)
		backup, err := os.OpenFile(backupPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return "", err
		}
		_, copyErr := io.Copy(backup, source)
		closeErr := backup.Close()
		if copyErr != nil {
			_ = os.Remove(backupPath)
			return "", copyErr
		}
		if closeErr != nil {
			_ = os.Remove(backupPath)
			return "", closeErr
		}
		return backupPath, nil
	}
}
