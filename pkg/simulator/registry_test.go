package simulator

import (
	"context"
	"testing"
)

// mockSimulator is a test implementation of Simulator
type mockSimulator struct {
	name     string
	category Category
}

func (m *mockSimulator) Name() string                                       { return m.name }
func (m *mockSimulator) Description() string                                { return "Mock simulator for testing" }
func (m *mockSimulator) Category() Category                                 { return m.category }
func (m *mockSimulator) Simulate(ctx context.Context, opts Options) (*Result, error) { return nil, nil }
func (m *mockSimulator) Cleanup(ctx context.Context) error                  { return nil }
func (m *mockSimulator) DryRun(opts Options) string                         { return "dry run" }
func (m *mockSimulator) IsReversible() bool                                 { return true }
func (m *mockSimulator) ShellCommand(opts Options) []string                 { return []string{"echo test"} }
func (m *mockSimulator) CleanupCommand() []string                           { return []string{"echo cleanup"} }

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry should not return nil")
	}
	if r.simulators == nil {
		t.Error("simulators map should be initialized")
	}
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()
	sim := &mockSimulator{name: "test-sim", category: CategoryKernel}

	err := r.Register(sim)
	if err != nil {
		t.Errorf("Register should not error: %v", err)
	}

	// Registering same name should error
	err = r.Register(sim)
	if err == nil {
		t.Error("Register should error for duplicate name")
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()
	sim := &mockSimulator{name: "test-sim", category: CategoryKernel}
	r.Register(sim)

	// Get existing
	got, ok := r.Get("test-sim")
	if !ok {
		t.Error("Get should return true for existing simulator")
	}
	if got.Name() != "test-sim" {
		t.Errorf("Get returned wrong simulator: %s", got.Name())
	}

	// Get non-existing
	_, ok = r.Get("non-existent")
	if ok {
		t.Error("Get should return false for non-existing simulator")
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockSimulator{name: "sim1", category: CategoryKernel})
	r.Register(&mockSimulator{name: "sim2", category: CategoryNetworking})

	list := r.List()
	if len(list) != 2 {
		t.Errorf("List should return 2 simulators, got %d", len(list))
	}
}

func TestRegistry_ListByCategory(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockSimulator{name: "kernel1", category: CategoryKernel})
	r.Register(&mockSimulator{name: "kernel2", category: CategoryKernel})
	r.Register(&mockSimulator{name: "network1", category: CategoryNetworking})

	kernelSims := r.ListByCategory(CategoryKernel)
	if len(kernelSims) != 2 {
		t.Errorf("ListByCategory(kernel) should return 2, got %d", len(kernelSims))
	}

	networkSims := r.ListByCategory(CategoryNetworking)
	if len(networkSims) != 1 {
		t.Errorf("ListByCategory(networking) should return 1, got %d", len(networkSims))
	}

	storageSims := r.ListByCategory(CategoryStorage)
	if len(storageSims) != 0 {
		t.Errorf("ListByCategory(storage) should return 0, got %d", len(storageSims))
	}
}

func TestCategories(t *testing.T) {
	// Verify all category constants are defined
	categories := []Category{
		CategoryKernel,
		CategoryNetworking,
		CategoryStorage,
		CategoryRuntime,
		CategoryAccelerator,
	}

	for _, c := range categories {
		if c == "" {
			t.Error("Category constant should not be empty")
		}
	}
}
