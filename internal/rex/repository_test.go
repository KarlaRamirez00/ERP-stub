package rex_test

import (
	"sync"
	"testing"

	"github.com/kramirez/rex-stub/internal/rex"
)

func TestRegisterAssignsSequentialReferencesPerKind(t *testing.T) {
	repo := rex.NewRepository()

	first, _ := repo.Register(rex.KindContract, "a")
	second, _ := repo.Register(rex.KindContract, "b")
	termination, _ := repo.Register(rex.KindTermination, "a")
	annex, _ := repo.Register(rex.KindAnnex, "a")

	expected := []string{"REX-000001", "REX-000002", "REX-TERM-000001", "REX-ANX-000001"}
	got := []string{first, second, termination, annex}
	for i := range expected {
		if got[i] != expected[i] {
			t.Fatalf("referencia %d: esperada %s, fue %s", i, expected[i], got[i])
		}
	}
}

func TestRegisterIsIdempotentByExternalID(t *testing.T) {
	repo := rex.NewRepository()

	first, created := repo.Register(rex.KindContract, "42")
	again, createdAgain := repo.Register(rex.KindContract, "42")

	if !created || createdAgain {
		t.Fatalf("created debía ser true y luego false, fue %v y %v", created, createdAgain)
	}
	if first != again {
		t.Fatalf("la referencia debía repetirse: %s vs %s", first, again)
	}
}

func TestExistsChecksKind(t *testing.T) {
	repo := rex.NewRepository()
	reference, _ := repo.Register(rex.KindContract, "1")

	if !repo.Exists(rex.KindContract, reference) {
		t.Fatal("el contrato registrado debía existir")
	}
	if repo.Exists(rex.KindTermination, reference) {
		t.Fatal("un contrato no debe contar como terminación")
	}
	if repo.Exists(rex.KindContract, "REX-999999") {
		t.Fatal("una referencia desconocida no debe existir")
	}
}

func TestRegisterIsSafeUnderConcurrency(t *testing.T) {
	repo := rex.NewRepository()
	const workers = 50

	var wg sync.WaitGroup
	references := make(chan string, workers)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reference, _ := repo.Register(rex.KindContract, "same-id")
			references <- reference
		}()
	}
	wg.Wait()
	close(references)

	unique := map[string]struct{}{}
	for reference := range references {
		unique[reference] = struct{}{}
	}
	if len(unique) != 1 {
		t.Fatalf("el mismo externalId debía dar una sola referencia, hubo %d", len(unique))
	}
}
