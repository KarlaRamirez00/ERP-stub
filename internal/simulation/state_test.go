package simulation_test

import (
	"testing"

	"github.com/kramirez/rex-stub/internal/simulation"
)

func TestSetRejectsInvalidSettingsAndKeepsPreviousState(t *testing.T) {
	state := simulation.NewState(simulation.Settings{LatencyMs: 10})

	fields := state.Set(simulation.Settings{LatencyMs: -1, FailureRate: 2})

	if len(fields) != 2 {
		t.Fatalf("se esperaban 2 campos inválidos, hubo %v", fields)
	}
	if state.Get().LatencyMs != 10 {
		t.Fatalf("el estado no debía cambiar, quedó %+v", state.Get())
	}
}

func TestSetReplacesWholeState(t *testing.T) {
	state := simulation.NewState(simulation.Settings{LatencyMs: 10, ForceError: true})

	if fields := state.Set(simulation.Settings{FailureRate: 0.5}); fields != nil {
		t.Fatalf("no se esperaban errores: %v", fields)
	}

	got := state.Get()
	if got.LatencyMs != 0 || got.ForceError || got.FailureRate != 0.5 {
		t.Fatalf("estado inesperado: %+v", got)
	}
}
