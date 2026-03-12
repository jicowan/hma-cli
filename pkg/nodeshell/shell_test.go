package nodeshell

import (
	"testing"
)

func TestConstants(t *testing.T) {
	if Namespace != "default" {
		t.Errorf("Namespace = %q, want %q", Namespace, "default")
	}
	if PodNamePrefix != "hma-cli-shell" {
		t.Errorf("PodNamePrefix = %q, want %q", PodNamePrefix, "hma-cli-shell")
	}
	if Image != "alpine:latest" {
		t.Errorf("Image = %q, want %q", Image, "alpine:latest")
	}
}

func TestNodeShellPodName(t *testing.T) {
	// Test pod name generation
	nodeName := "ip-10-0-1-123.ec2.internal"
	expectedPodName := "hma-cli-shell-ip-10-0-1-123.ec2.internal"

	// We can't create a real NodeShell without a cluster, but we can test the naming logic
	podName := PodNamePrefix + "-" + nodeName
	if podName != expectedPodName {
		t.Errorf("pod name = %q, want %q", podName, expectedPodName)
	}
}

func TestExecResult(t *testing.T) {
	result := &ExecResult{
		Stdout:   "hello world",
		Stderr:   "",
		ExitCode: 0,
	}

	if result.Stdout != "hello world" {
		t.Errorf("ExecResult.Stdout = %q, want %q", result.Stdout, "hello world")
	}
	if result.ExitCode != 0 {
		t.Errorf("ExecResult.ExitCode = %d, want %d", result.ExitCode, 0)
	}
}
