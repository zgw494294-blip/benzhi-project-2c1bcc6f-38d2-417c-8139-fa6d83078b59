package domain

import (
	"errors"
	"strings"
)

func ValidateMetadata(m map[string]string) error {
	if len(m) == 0 {
		return errors.New("必须提供馆藏元数据")
	}
	for k, v := range m {
		if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
			return errors.New("元数据键和值不能为空")
		}
	}
	return nil
}
func ValidateRoleActor(actor string, role Role) error {
	if strings.TrimSpace(actor) == "" {
		return errors.New("actor 不能为空")
	}
	switch role {
	case RoleArchivist, RoleConservator, RoleReviewer:
		return nil
	default:
		return errors.New("未知角色")
	}
}
