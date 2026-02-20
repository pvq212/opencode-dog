package provider

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"testing"
)

// --- mock provider for registry tests ---

type mockProvider struct {
	typ ProviderType
}

func (m *mockProvider) Type() ProviderType { return m.typ }
func (m *mockProvider) ValidateConfig(_ map[string]any) error {
	return nil
}
func (m *mockProvider) BuildHandler(_ string, _ string, _ map[string]any, _ func(context.Context, *IncomingMessage)) http.Handler {
	return nil
}
func (m *mockProvider) SendReply(_ context.Context, _ map[string]any, _ *IncomingMessage, _ string) error {
	return nil
}

// --- NewRegistry ---

func TestNewRegistry(t *testing.T) {
	logger := slog.Default()
	reg := NewRegistry(logger)
	if reg == nil {
		t.Fatal("NewRegistry() returned nil")
	}
	if reg.providers == nil {
		t.Fatal("NewRegistry() providers map is nil")
	}
	if len(reg.providers) != 0 {
		t.Errorf("NewRegistry() providers count = %d, want 0", len(reg.providers))
	}
}

// --- Register ---

func TestRegister(t *testing.T) {
	reg := NewRegistry(slog.Default())
	p := &mockProvider{typ: "test"}
	reg.Register(p)

	if len(reg.providers) != 1 {
		t.Fatalf("providers count = %d, want 1", len(reg.providers))
	}

	got, ok := reg.providers["test"]
	if !ok {
		t.Fatal("provider 'test' not found in map")
	}
	if got != p {
		t.Error("registered provider does not match")
	}
}

func TestRegister_Overwrite(t *testing.T) {
	reg := NewRegistry(slog.Default())
	p1 := &mockProvider{typ: "test"}
	p2 := &mockProvider{typ: "test"}
	reg.Register(p1)
	reg.Register(p2)

	if len(reg.providers) != 1 {
		t.Fatalf("providers count = %d, want 1", len(reg.providers))
	}
	got, _ := reg.providers["test"]
	if got != p2 {
		t.Error("second register should overwrite first")
	}
}

// --- Get ---

func TestGet_Found(t *testing.T) {
	reg := NewRegistry(slog.Default())
	p := &mockProvider{typ: "gitlab"}
	reg.Register(p)

	got, ok := reg.Get("gitlab")
	if !ok {
		t.Fatal("Get() ok = false, want true")
	}
	if got != p {
		t.Error("Get() returned wrong provider")
	}
}

func TestGet_NotFound(t *testing.T) {
	reg := NewRegistry(slog.Default())

	got, ok := reg.Get("nonexistent")
	if ok {
		t.Fatal("Get() ok = true, want false")
	}
	if got != nil {
		t.Error("Get() returned non-nil for missing provider")
	}
}

// --- All ---

func TestAll_Empty(t *testing.T) {
	reg := NewRegistry(slog.Default())
	all := reg.All()
	if len(all) != 0 {
		t.Errorf("All() len = %d, want 0", len(all))
	}
}

func TestAll_ReturnsCopy(t *testing.T) {
	reg := NewRegistry(slog.Default())
	reg.Register(&mockProvider{typ: "a"})
	reg.Register(&mockProvider{typ: "b"})

	all := reg.All()
	if len(all) != 2 {
		t.Fatalf("All() len = %d, want 2", len(all))
	}

	all["c"] = &mockProvider{typ: "c"}

	if len(reg.providers) != 2 {
		t.Error("modifying All() result should not affect registry")
	}
}

func TestAll_ContainsAllProviders(t *testing.T) {
	reg := NewRegistry(slog.Default())
	reg.Register(&mockProvider{typ: "x"})
	reg.Register(&mockProvider{typ: "y"})

	all := reg.All()
	if _, ok := all["x"]; !ok {
		t.Error("All() missing provider 'x'")
	}
	if _, ok := all["y"]; !ok {
		t.Error("All() missing provider 'y'")
	}
}

// --- Concurrent access ---

func TestConcurrentAccess(t *testing.T) {
	reg := NewRegistry(slog.Default())
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(3)
		typ := ProviderType("provider-" + string(rune('a'+i%26)))

		go func() {
			defer wg.Done()
			reg.Register(&mockProvider{typ: typ})
		}()

		go func() {
			defer wg.Done()
			reg.Get(typ)
		}()

		go func() {
			defer wg.Done()
			reg.All()
		}()
	}

	wg.Wait()
}
