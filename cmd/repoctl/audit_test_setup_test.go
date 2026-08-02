package main

import "repoctl/internal/apply"

func init() {
	newAudit = func(string) (*apply.Audit, error) {
		return nil, nil
	}
}
