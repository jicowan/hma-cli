package diagnose

import (
	"testing"
)

func TestGVR(t *testing.T) {
	// Verify the GVR is correctly defined
	if gvr.Group != "eks.amazonaws.com" {
		t.Errorf("GVR group = %q, want %q", gvr.Group, "eks.amazonaws.com")
	}
	if gvr.Version != "v1alpha1" {
		t.Errorf("GVR version = %q, want %q", gvr.Version, "v1alpha1")
	}
	if gvr.Resource != "nodediagnostics" {
		t.Errorf("GVR resource = %q, want %q", gvr.Resource, "nodediagnostics")
	}
}

func TestConstants(t *testing.T) {
	if apiGroup != "eks.amazonaws.com" {
		t.Errorf("apiGroup = %q, want %q", apiGroup, "eks.amazonaws.com")
	}
	if apiVersion != "v1alpha1" {
		t.Errorf("apiVersion = %q, want %q", apiVersion, "v1alpha1")
	}
	if resource != "nodediagnostics" {
		t.Errorf("resource = %q, want %q", resource, "nodediagnostics")
	}
}

func TestStatus(t *testing.T) {
	status := &Status{
		Phase:   "Success",
		Message: "Log collection complete",
	}

	if status.Phase != "Success" {
		t.Errorf("Status.Phase = %q, want %q", status.Phase, "Success")
	}
	if status.Message != "Log collection complete" {
		t.Errorf("Status.Message = %q, want %q", status.Message, "Log collection complete")
	}
}
