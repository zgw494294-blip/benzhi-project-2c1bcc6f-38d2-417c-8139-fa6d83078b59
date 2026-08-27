package persistence

import (
	"archive-release/internal/domain"
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
)

func AppendJSONLEvent(dir string, e domain.Event) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(dir, "events.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	_, err = f.Write(append(b, '\n'))
	return err
}
func ReadJSONLEvents(dir string) ([]domain.Event, error) {
	f, err := os.Open(filepath.Join(dir, "events.jsonl"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []domain.Event
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var e domain.Event
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, sc.Err()
}
