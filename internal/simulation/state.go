package simulation

import "sync"

type Settings struct {
	LatencyMs   int     `json:"latencyMs"`
	FailureRate float64 `json:"failureRate"`
	ForceError  bool    `json:"forceError"`
}

func (s Settings) Validate() map[string]string {
	fields := map[string]string{}
	if s.LatencyMs < 0 {
		fields["latencyMs"] = "no puede ser negativo"
	}
	if s.FailureRate < 0 || s.FailureRate > 1 {
		fields["failureRate"] = "debe estar entre 0 y 1"
	}
	if len(fields) == 0 {
		return nil
	}
	return fields
}

type State struct {
	mu       sync.RWMutex
	settings Settings
}

func NewState(settings Settings) *State {
	return &State{settings: settings}
}

func (s *State) Get() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

func (s *State) Set(settings Settings) map[string]string {
	if fields := settings.Validate(); fields != nil {
		return fields
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings = settings
	return nil
}
