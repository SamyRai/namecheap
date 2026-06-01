package client

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// AuditEvent represents a single mutating operation
type AuditEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Account   string    `json:"account"`
	Provider  string    `json:"provider"`
	Operation string    `json:"operation"`
	Target    string    `json:"target"`
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
}

// LogAuditEvent records a mutating operation to the audit log
func LogAuditEvent(account, provider, operation, target string, err error) {
	status := "SUCCESS"
	var errStr string
	if err != nil {
		status = "ERROR"
		errStr = err.Error()
	}

	event := AuditEvent{
		Timestamp: time.Now().UTC(),
		Account:   account,
		Provider:  provider,
		Operation: operation,
		Target:    target,
		Status:    status,
		Error:     errStr,
	}

	home, dirErr := os.UserHomeDir()
	if dirErr != nil {
		return // Silently fail if we can't find home dir
	}

	logDir := filepath.Join(home, ".zonekit")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return
	}

	logFile := filepath.Join(logDir, "audit.log")
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	b, jsonErr := json.Marshal(event)
	if jsonErr != nil {
		return
	}

	fmt.Fprintf(f, "%s\n", string(b))
}
