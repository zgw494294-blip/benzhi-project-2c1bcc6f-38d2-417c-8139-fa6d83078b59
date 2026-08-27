package persistence

import "sync"

type CommandLog struct {
	mu     sync.RWMutex
	values map[string]any
}

func NewCommandLog() *CommandLog { return &CommandLog{values: map[string]any{}} }
func (l *CommandLog) Get(k string) (any, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	v, ok := l.values[k]
	return v, ok
}
func (l *CommandLog) Put(k string, v any) { l.mu.Lock(); defer l.mu.Unlock(); l.values[k] = v }
func (l *CommandLog) Len() int            { l.mu.RLock(); defer l.mu.RUnlock(); return len(l.values) }
