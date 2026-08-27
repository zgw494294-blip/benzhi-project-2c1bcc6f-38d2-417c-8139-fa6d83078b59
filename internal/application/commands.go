package application

import (
	"archive-release/internal/domain"
	"errors"
	"fmt"
)

type CreateRequest struct {
	Title       string
	ArchiveUnit string
	Creator     string
	Actor       string
	Role        domain.Role
	Key         string
}

func (r CreateRequest) Validate() error {
	if err := CheckCommand(r.Actor, r.Role, r.Key); err != nil {
		return err
	}
	return domain.ValidateCaseInput(r.Title, r.ArchiveUnit, r.Creator)
}

type RevisionRequest struct {
	Actor    string
	Role     domain.Role
	Expected int
	Key      string
	Parent   string
	Pages    []domain.PageMeasurement
	Metadata map[string]string
}

func (r RevisionRequest) Validate() error {
	if err := CheckCommand(r.Actor, r.Role, r.Key); err != nil {
		return err
	}
	if r.Expected < 1 {
		return errors.New("expectedVersion 必须为正数")
	}
	if len(r.Pages) == 0 {
		return errors.New("修订页面不能为空")
	}
	return domain.ValidateMetadata(r.Metadata)
}
func (s *Service) CommandSummary(c *domain.ArchiveCase) string {
	if c == nil {
		return ""
	}
	return fmt.Sprintf("%s/%s v%d", c.CaseID, c.Status, c.Version)
}
