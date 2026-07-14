package bookmarks

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func DefaultStorePath() string {
	if path := strings.TrimSpace(os.Getenv("BRMK_DATA")); path != "" {
		return path
	}
	if dir := defaultConfigDir(); dir != "" {
		return filepath.Join(dir, "brmk", "bookmarks.json")
	}
	return "bookmarks.json"
}

func Load(path string) (*Store, error) {
	if path == "" {
		path = DefaultStorePath()
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return NewStore(), nil
	}
	if err != nil {
		return nil, err
	}
	return loadStoreData(data)
}

func loadStoreData(data []byte) (*Store, error) {
	if len(data) == 0 {
		return NewStore(), nil
	}
	var store Store
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	return Normalize(&store), nil
}

func Save(path string, store *Store) error {
	if path == "" {
		path = DefaultStorePath()
	}
	store = Normalize(store)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := BackupStoreFile(path); err != nil {
		return err
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func BackupStoreFile(path string) error {
	if path == "" {
		path = DefaultStorePath()
	}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.IsDir() || info.Size() == 0 {
		return nil
	}
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()

	dir := filepath.Dir(path)
	backupDir := filepath.Join(dir, "backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return err
	}
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	backupName := fmt.Sprintf("%s-%s.json", name, time.Now().UTC().Format("20060102T150405.000000000Z"))
	dst, err := os.OpenFile(filepath.Join(backupDir, backupName), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		return err
	}
	if err := dst.Close(); err != nil {
		return err
	}
	return pruneStoreBackups(backupDir, name, 50)
}

func pruneStoreBackups(dir, name string, keep int) error {
	if keep <= 0 {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	prefix := name + "-"
	var backups []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		entryName := entry.Name()
		if strings.HasPrefix(entryName, prefix) && strings.HasSuffix(entryName, ".json") {
			backups = append(backups, filepath.Join(dir, entryName))
		}
	}
	sort.Strings(backups)
	if len(backups) <= keep {
		return nil
	}
	for _, path := range backups[:len(backups)-keep] {
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	return nil
}
