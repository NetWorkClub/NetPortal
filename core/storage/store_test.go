package storage

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/NetWorkClub/NetPortal/core/api"
)

func TestStorePersistenceAndCascadeDelete(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "netportal", "state.json")

	st, err := Open(statePath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	if err := st.UpsertSource(api.SourceItem{
		ID:   "src-1",
		Name: "AWG Home",
		Meta: "Local .conf",
	}); err != nil {
		t.Fatalf("UpsertSource failed: %v", err)
	}

	if err := st.UpsertNodes([]api.NodeItem{
		{
			ID:         "node-1",
			SourceID:   "src-1",
			Name:       "AWG 3.1",
			Endpoint:   "198.51.100.1:51820",
			Proto:      "AWG 3.1",
			ProtoGroup: api.ProtoGroupAWG,
			MTU:        1280,
			Params:     map[string]string{"Jc": "5"},
		},
	}); err != nil {
		t.Fatalf("UpsertNodes failed: %v", err)
	}

	prof, err := st.CreateProfile("prof-1", "Профиль 1", "Profile 1", "direct")
	if err != nil {
		t.Fatalf("CreateProfile failed: %v", err)
	}
	if len(prof.Rules) != 0 {
		t.Fatalf("expected empty rules on new profile, got %d", len(prof.Rules))
	}

	reopened, err := Open(statePath)
	if err != nil {
		t.Fatalf("reopen failed: %v", err)
	}
	if len(reopened.ListSources()) != 1 || len(reopened.ListNodes()) != 1 || len(reopened.ListProfiles()) != 1 {
		t.Fatalf("unexpected counts after reopen: %+v", reopened.Snapshot())
	}

	if err := reopened.DeleteSource("src-1"); err != nil {
		t.Fatalf("DeleteSource failed: %v", err)
	}
	if len(reopened.ListSources()) != 0 || len(reopened.ListNodes()) != 0 {
		t.Fatalf("expected cascade delete of source nodes")
	}
}

func TestStoreConcurrentWrites(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	st, err := Open(statePath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			id := string(rune('a' + idx))
			_ = st.UpsertSource(api.SourceItem{ID: "src-" + id, Name: id})
		}(i)
	}
	wg.Wait()

	if len(st.ListSources()) != 16 {
		t.Fatalf("expected 16 sources, got %d", len(st.ListSources()))
	}
}
