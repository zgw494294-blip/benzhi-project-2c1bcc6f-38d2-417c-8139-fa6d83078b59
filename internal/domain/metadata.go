package domain

import (
	"sort"
	"strings"
)

func NormalizeText(v string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(v)), " "))
}

type MetadataField struct {
	Key        string `json:"key"`
	Value      string `json:"value"`
	Normalized string `json:"normalized"`
}

func NormalizeMetadata(m map[string]string) []MetadataField {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]MetadataField, 0, len(keys))
	for _, k := range keys {
		out = append(out, MetadataField{Key: k, Value: m[k], Normalized: strings.ToLower(strings.TrimSpace(m[k]))})
	}
	return out
}
func MetadataDigest(m map[string]string) string { return Digest(NormalizeMetadata(m)) }
func MergeMetadata(base, override map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range override {
		out[k] = v
	}
	return out
}
