package persistence

import (
	"archive-release/internal/domain"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

type Repository interface {
	Create(*domain.ArchiveCase) error
	List() []*domain.ArchiveCase
	Get(string) (*domain.ArchiveCase, error)
	Save(*domain.ArchiveCase, domain.Event) error
	Events(string) []domain.Event
	FindCommand(string) (any, bool)
	SetCommand(string, any)
	Ready() error
}
type MetadataFinder interface {
	FindMetadata(string) (*domain.ArchiveCase, bool)
}
type MemoryRepository struct {
	mu       sync.RWMutex
	cases    map[string]*domain.ArchiveCase
	events   map[string][]domain.Event
	commands map[string]any
}

func NewMemory() *MemoryRepository {
	return &MemoryRepository{cases: map[string]*domain.ArchiveCase{}, events: map[string][]domain.Event{}, commands: map[string]any{}}
}
func (r *MemoryRepository) Create(c *domain.ArchiveCase) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.cases[c.CaseID]; ok {
		return errors.New("案卷已存在")
	}
	r.cases[c.CaseID] = clone(c)
	return nil
}
func (r *MemoryRepository) List() []*domain.ArchiveCase {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*domain.ArchiveCase, 0, len(r.cases))
	for _, c := range r.cases {
		out = append(out, clone(c))
	}
	return out
}
func (r *MemoryRepository) Get(id string) (*domain.ArchiveCase, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.cases[id]
	if !ok {
		return nil, os.ErrNotExist
	}
	return clone(c), nil
}
func (r *MemoryRepository) Save(c *domain.ArchiveCase, e domain.Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.cases[c.CaseID]; !ok {
		return os.ErrNotExist
	}
	r.cases[c.CaseID] = clone(c)
	r.events[c.CaseID] = append(r.events[c.CaseID], e)
	return nil
}
func (r *MemoryRepository) Events(id string) []domain.Event {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.events[id]
}
func (r *MemoryRepository) FindCommand(k string) (any, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.commands[k]
	return v, ok
}
func (r *MemoryRepository) SetCommand(k string, v any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.commands[k] = v
}
func (r *MemoryRepository) FindMetadata(digest string) (*domain.ArchiveCase, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.cases {
		if digest != "" && c.MetadataDigest == digest {
			return clone(c), true
		}
	}
	return nil, false
}
func (r *MemoryRepository) Ready() error { return nil }
func clone(c *domain.ArchiveCase) *domain.ArchiveCase {
	b, _ := json.Marshal(c)
	var x domain.ArchiveCase
	_ = json.Unmarshal(b, &x)
	return &x
}

type FileRepository struct {
	*MemoryRepository
	dir string
}

func NewFile(dir string) (*FileRepository, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	r := &FileRepository{MemoryRepository: NewMemory(), dir: dir}
	if err := r.load(); err != nil {
		return nil, err
	}
	return r, nil
}
func (r *FileRepository) load() error {
	b, err := os.ReadFile(filepath.Join(r.dir, "cases.json"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var cs []*domain.ArchiveCase
	if err = json.Unmarshal(b, &cs); err != nil {
		return errors.New("持久化记录损坏")
	}
	for _, c := range cs {
		if c.NormalizedTitle == "" {
			c.NormalizedTitle = domain.NormalizeText(c.Title)
		}
		if c.NormalizedUnit == "" {
			c.NormalizedUnit = domain.NormalizeText(c.ArchiveUnit)
		}
		if c.NormalizedCreator == "" {
			c.NormalizedCreator = domain.NormalizeText(c.Creator)
		}
		if c.MetadataDigest == "" {
			c.MetadataDigest = domain.Digest(struct{ Unit, Title string }{c.NormalizedUnit, c.NormalizedTitle})
		}
		r.cases[c.CaseID] = c
	}
	return nil
}
func (r *FileRepository) persist() error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cs := make([]*domain.ArchiveCase, 0, len(r.cases))
	for _, c := range r.cases {
		cs = append(cs, c)
	}
	b, _ := json.MarshalIndent(cs, "", "  ")
	tmp := filepath.Join(r.dir, "cases.tmp")
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(r.dir, "cases.json"))
}
func (r *FileRepository) Create(c *domain.ArchiveCase) error {
	if err := r.MemoryRepository.Create(c); err != nil {
		return err
	}
	return r.persist()
}
func (r *FileRepository) Save(c *domain.ArchiveCase, e domain.Event) error {
	if err := r.MemoryRepository.Save(c, e); err != nil {
		return err
	}
	return r.persist()
}
