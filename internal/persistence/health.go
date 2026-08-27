package persistence

import "errors"

type Health struct {
	Ready   bool
	Message string
}

func Check(repo Repository) Health {
	if err := repo.Ready(); err != nil {
		return Health{Message: err.Error()}
	}
	return Health{Ready: true, Message: "仓储就绪"}
}
func RequireReady(repo Repository) error {
	h := Check(repo)
	if !h.Ready {
		return errors.New(h.Message)
	}
	return nil
}
