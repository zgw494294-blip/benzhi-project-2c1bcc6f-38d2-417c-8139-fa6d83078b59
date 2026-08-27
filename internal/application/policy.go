package application

import (
	"archive-release/internal/domain"
	"errors"
)

func CheckCommand(actor string, role domain.Role, key string) error {
	if err := domain.ValidateRoleActor(actor, role); err != nil {
		return err
	}
	if key == "" {
		return errors.New("idempotencyKey 不能为空")
	}
	return nil
}
func CheckExpected(actual, want int) error {
	if actual != want {
		return errors.New("版本冲突")
	}
	return nil
}
