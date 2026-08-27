package domain

import "time"

type Event struct {
	Seq        int64     `json:"seq"`
	Type       string    `json:"type"`
	CaseID     string    `json:"caseID"`
	Actor      string    `json:"actor"`
	Role       Role      `json:"role"`
	CommandKey string    `json:"commandKey"`
	Version    int       `json:"version,omitempty"`
	At         time.Time `json:"at"`
	Payload    any       `json:"payload"`
	PrevDigest string    `json:"prevDigest"`
	Digest     string    `json:"digest"`
}

func NewEvent(seq int64, typ, caseID, actor string, role Role, key string, at time.Time, payload any, prev string) Event {
	e := Event{Seq: seq, Type: typ, CaseID: caseID, Actor: actor, Role: role, CommandKey: key, At: at, Payload: payload, PrevDigest: prev}
	e.Digest = Digest(e)
	return e
}
