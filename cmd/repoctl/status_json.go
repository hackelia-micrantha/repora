package main

import "encoding/json"

const (
	statusOutputKind    = "repora.status"
	statusOutputVersion = 1
)

// MarshalJSON adds the stable envelope without changing the existing status
// payload model.
func (output jsonOutput) MarshalJSON() ([]byte, error) {
	type statusDocument struct {
		Kind    string     `json:"kind"`
		Version int        `json:"version"`
		Repos   []jsonRepo `json:"repos"`
	}
	return json.Marshal(statusDocument{
		Kind:    statusOutputKind,
		Version: statusOutputVersion,
		Repos:   output.Repos,
	})
}
