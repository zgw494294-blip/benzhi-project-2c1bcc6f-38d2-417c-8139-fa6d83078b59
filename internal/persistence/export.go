package persistence

import (
	"archive-release/internal/domain"
	"encoding/json"
	"os"
	"path/filepath"
)

func ExportCase(repo Repository, id, path string) error {
	c, e := repo.Get(id)
	if e != nil {
		return e
	}
	b, e := json.MarshalIndent(c, "", "  ")
	if e != nil {
		return e
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}
func ImportCase(repo Repository, path string) error {
	b, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	c, e := domain.UnmarshalCase(b)
	if e != nil {
		return e
	}
	return repo.Create(c)
}
