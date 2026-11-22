package rex

import (
	"fmt"
	"sync"
)

type Kind string

const (
	KindContract    Kind = "contract"
	KindTermination Kind = "termination"
	KindAnnex       Kind = "annex"
)

var referencePrefixes = map[Kind]string{
	KindContract:    "REX-",
	KindTermination: "REX-TERM-",
	KindAnnex:       "REX-ANX-",
}

type Repository struct {
	mu           sync.RWMutex
	counters     map[Kind]int
	byExternalID map[Kind]map[string]string
	kindByRef    map[string]Kind
}

func NewRepository() *Repository {
	byExternalID := make(map[Kind]map[string]string, len(referencePrefixes))
	for kind := range referencePrefixes {
		byExternalID[kind] = map[string]string{}
	}
	return &Repository{
		counters:     map[Kind]int{},
		byExternalID: byExternalID,
		kindByRef:    map[string]Kind{},
	}
}

func (r *Repository) Register(kind Kind, externalID string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if reference, found := r.byExternalID[kind][externalID]; found {
		return reference, false
	}
	r.counters[kind]++
	reference := fmt.Sprintf("%s%06d", referencePrefixes[kind], r.counters[kind])
	r.byExternalID[kind][externalID] = reference
	r.kindByRef[reference] = kind
	return reference, true
}

func (r *Repository) Exists(kind Kind, reference string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	found, ok := r.kindByRef[reference]
	return ok && found == kind
}
