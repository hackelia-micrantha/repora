package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path/filepath"

	"repoctl/internal/apply"
	"repoctl/internal/journal"
)

var newAudit = defaultAudit

func defaultAudit(configPath string) (*apply.Audit, error) {
	executionID, writer, err := newJournalContext(configPath)
	if err != nil {
		return nil, err
	}
	return &apply.Audit{ExecutionID: executionID, Writer: writer}, nil
}

func newJournalContext(configPath string) (string, journal.Writer, error) {
	executionID, err := newExecutionID()
	if err != nil {
		return "", journal.Writer{}, err
	}
	absoluteConfig, err := filepath.Abs(configPath)
	if err != nil {
		return "", journal.Writer{}, fmt.Errorf("resolve config path for journal root: %w", err)
	}
	return executionID, journal.Writer{Root: filepath.Dir(absoluteConfig)}, nil
}

func newExecutionID() (string, error) {
	var entropy [16]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		return "", fmt.Errorf("generate execution ID: %w", err)
	}
	return "run-" + hex.EncodeToString(entropy[:]), nil
}
