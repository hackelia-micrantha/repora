package main

import "encoding/json"

const (
	statusOutputKind    = "repora.status"
	statusOutputVersion = 1
)

// MarshalJSON adds the stable contract envelope while preserving the existing
// status payload fields for compatibility with current consumers.
func (output jsonOutput) MarshalJSON() ([]byte, error) {
	type outputAlias jsonOutput
	return json.Marshal(struct {
		Kind    string `json:"kind"`
		Version int    `json:"version"`
		outputAlias
	}{
		Kind:        statusOutputKind,
		Version:     statusOutputVersion,
		outputAlias: outputAlias(output),
	})
}
