package domain

import "time"

type AuditEntry struct {
	Seq     int64     `json:"seq"`
	Action  string    `json:"action"`
	Actor   string    `json:"actor"`
	Role    Role      `json:"role"`
	At      time.Time `json:"at"`
	Version int       `json:"version"`
	Summary string    `json:"summary"`
}

func EventAudit(e Event) AuditEntry {
	return AuditEntry{Seq: e.Seq, Action: e.Type, Actor: e.Actor, Role: e.Role, At: e.At, Version: e.Version, Summary: e.CommandKey}
}
func AuditTrail(events []Event) []AuditEntry {
	out := make([]AuditEntry, 0, len(events))
	for _, e := range events {
		out = append(out, EventAudit(e))
	}
	return out
}
