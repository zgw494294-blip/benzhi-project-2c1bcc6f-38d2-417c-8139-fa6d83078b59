package domain

import "errors"

func (c *ArchiveCase) CanSubmitRevision() error {
	if c.Status == StatusFrozen {
		return errors.New("案卷已冻结")
	}
	return nil
}
func (c *ArchiveCase) CanReview() error {
	if c.Status != StatusReviewPending {
		return errors.New("案卷尚未达到待复核状态")
	}
	if c.BlockingOpen() > 0 {
		return errors.New("仍有阻断发现")
	}
	return nil
}
func (c *ArchiveCase) CanFreeze() error {
	if c.Status != StatusApproved {
		return errors.New("案卷未获批准")
	}
	if c.BlockingOpen() > 0 {
		return errors.New("仍有阻断发现")
	}
	return nil
}
func (c *ArchiveCase) Transition(next CaseStatus) error {
	if c.Status == StatusFrozen && next != StatusFrozen {
		return errors.New("冻结后不能改变状态")
	}
	c.Status = next
	return nil
}
