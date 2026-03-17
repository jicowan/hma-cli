package aws

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateLogKey(t *testing.T) {
	nodeName := "ip-10-0-1-123.ec2.internal"
	key := GenerateLogKey(nodeName)

	// Check format: timestamp/node-name/logs.tar.gz
	if !strings.HasSuffix(key, "/logs.tar.gz") {
		t.Errorf("key should end with /logs.tar.gz, got: %s", key)
	}

	if !strings.Contains(key, nodeName) {
		t.Errorf("key should contain node name %s, got: %s", nodeName, key)
	}

	// Check timestamp format (should be like 2026-03-17T15-04-05Z)
	parts := strings.Split(key, "/")
	if len(parts) != 3 {
		t.Errorf("key should have 3 parts (timestamp/node/file), got %d: %s", len(parts), key)
	}

	timestamp := parts[0]
	_, err := time.Parse("2006-01-02T15-04-05Z", timestamp)
	if err != nil {
		t.Errorf("timestamp %s is not in expected format: %v", timestamp, err)
	}
}

func TestGenerateLogKey_Format(t *testing.T) {
	tests := []struct {
		nodeName string
	}{
		{"ip-10-0-1-123.ec2.internal"},
		{"my-node"},
		{"node-with-special.chars"},
	}

	for _, tt := range tests {
		t.Run(tt.nodeName, func(t *testing.T) {
			key := GenerateLogKey(tt.nodeName)

			// Should always end with /logs.tar.gz
			if !strings.HasSuffix(key, "/logs.tar.gz") {
				t.Errorf("key should end with /logs.tar.gz, got: %s", key)
			}

			// Should contain node name
			if !strings.Contains(key, tt.nodeName) {
				t.Errorf("key should contain node name, got: %s", key)
			}
		})
	}
}
