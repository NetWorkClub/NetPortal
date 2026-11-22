// Package storage provides thread-safe, atomic local persistence for
// NetPortal sources, parsed nodes, routing profiles, and application settings.
package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/NetWorkClub/NetPortal/core/api"
)

const currentSchemaVersion = 1

var ErrNotFound = errors.New("storage: item not found")

// State represents the persisted JSON document on disk.
type State struct {
	Version  int                  `json:"version"`
	Sources  []api.SourceItem     `json:"sources"`
	Nodes    []api.NodeItem       `json:"nodes"`
	Profiles []api.RoutingProfile `json:"profiles"`
	Settings api.AppSettings      `json:"settings"`
}

// Store manages thread-safe reads and atomic writes to a local JSON state file.
type Store struct {
	mu    sync.RWMutex
	path  string
	state State
}

// Open loads an existing store from path or initializes a clean empty store.
func Open(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("storage: path must not be empty")
	}
	s := &Store{
		path:  path,
		state: defaultState(),
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return s, nil
		}
		return nil, fmt.Errorf("storage: read %s: %w", path, err)
	}
	if len(data) == 0 {
		return s, nil
	}
	var loaded State
	if err := json.Unmarshal(data, &loaded); err != nil {
		return nil, fmt.Errorf("storage: decode %s: %w", path, err)
	}
	normalizeState(&loaded)
	s.state = loaded
	return s, nil
}

func defaultState() State {
	return State{
		Version:  currentSchemaVersion,
		Sources:  []api.SourceItem{},
		Nodes:    []api.NodeItem{},
		Profiles: []api.RoutingProfile{},
		Settings: api.AppSettings{
			Theme:        "dark",
			Lang:         "ru",
			ServiceMode:  string(api.ServiceModeVPN),
			SmartDnsMode: string(api.SmartDnsProxy),
			Ipv6Mode:     "ipv4-only",
			Balancer:     "sticky",
		},
	}
}

func normalizeState(st *State) {
	if st.Version == 0 {
		st.Version = currentSchemaVersion
	}
	if st.Sources == nil {
		st.Sources = []api.SourceItem{}
	}
	if st.Nodes == nil {
		st.Nodes = []api.NodeItem{}
	}
	if st.Profiles == nil {
		st.Profiles = []api.RoutingProfile{}
	}
	for i := range st.Profiles {
		if st.Profiles[i].Rules == nil {
			st.Profiles[i].Rules = []api.RoutingRule{}
		}
	}
	for i := range st.Nodes {
		if st.Nodes[i].Params == nil {
			st.Nodes[i].Params = map[string]string{}
		}
	}
}

func (s *Store) persistLocked() error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("storage: mkdir %s: %w", dir, err)
	}
	payload, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return fmt.Errorf("storage: marshal: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".netportal-state-*.tmp")
	if err != nil {
		return fmt.Errorf("storage: create temp: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(payload); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("storage: write temp: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("storage: chmod temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("storage: close temp: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("storage: rename temp: %w", err)
	}
	cleanup = false
	return nil
}

// Snapshot returns a deep copy of the current state.
func (s *Store) Snapshot() State {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := s.state
	out.Sources = append([]api.SourceItem(nil), s.state.Sources...)
	out.Nodes = make([]api.NodeItem, len(s.state.Nodes))
	for i, n := range s.state.Nodes {
		out.Nodes[i] = cloneNode(n)
	}
	out.Profiles = make([]api.RoutingProfile, len(s.state.Profiles))
	for i, p := range s.state.Profiles {
		out.Profiles[i] = cloneProfile(p)
	}
	return out
}

func cloneNode(n api.NodeItem) api.NodeItem {
	cp := n
	if n.TcpMs != nil {
		v := *n.TcpMs
		cp.TcpMs = &v
	}
	if n.UrlMs != nil {
		v := *n.UrlMs
		cp.UrlMs = &v
	}
	cp.Params = make(map[string]string, len(n.Params))
	for k, v := range n.Params {
		cp.Params[k] = v
	}
	return cp
}

func cloneProfile(p api.RoutingProfile) api.RoutingProfile {
	cp := p
	cp.Rules = append([]api.RoutingRule(nil), p.Rules...)
	return cp
}

// GetSettings returns current user preferences.
func (s *Store) GetSettings() api.AppSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.Settings
}

// SaveSettings updates and persists user preferences.
func (s *Store) SaveSettings(settings api.AppSettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Settings = settings
	return s.persistLocked()
}

// ListSources returns all persisted sources.
func (s *Store) ListSources() []api.SourceItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]api.SourceItem(nil), s.state.Sources...)
}

// UpsertSource inserts or updates a source by ID.
func (s *Store) UpsertSource(src api.SourceItem) error {
	if src.ID == "" {
		return errors.New("storage: source ID must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.state.Sources {
		if existing.ID == src.ID {
			s.state.Sources[i] = src
			return s.persistLocked()
		}
	}
	s.state.Sources = append(s.state.Sources, src)
	return s.persistLocked()
}

// DeleteSource removes a source and all nodes belonging to it.
func (s *Store) DeleteSource(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := -1
	for i, src := range s.state.Sources {
		if src.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return ErrNotFound
	}
	s.state.Sources = append(s.state.Sources[:idx], s.state.Sources[idx+1:]...)
	filtered := s.state.Nodes[:0]
	for _, n := range s.state.Nodes {
		if n.SourceID != id {
			filtered = append(filtered, n)
		}
	}
	s.state.Nodes = filtered
	return s.persistLocked()
}

// ListNodes returns copies of all stored nodes.
func (s *Store) ListNodes() []api.NodeItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]api.NodeItem, len(s.state.Nodes))
	for i, n := range s.state.Nodes {
		out[i] = cloneNode(n)
	}
	return out
}

// GetNode fetches a single node by ID.
func (s *Store) GetNode(id string) (api.NodeItem, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, n := range s.state.Nodes {
		if n.ID == id {
			return cloneNode(n), true
		}
	}
	return api.NodeItem{}, false
}

// UpsertNodes inserts or updates multiple nodes atomically.
func (s *Store) UpsertNodes(nodes []api.NodeItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, node := range nodes {
		if node.ID == "" {
			return errors.New("storage: node ID must not be empty")
		}
		if node.Params == nil {
			node.Params = map[string]string{}
		}
		replaced := false
		for i, existing := range s.state.Nodes {
			if existing.ID == node.ID {
				s.state.Nodes[i] = cloneNode(node)
				replaced = true
				break
			}
		}
		if !replaced {
			s.state.Nodes = append(s.state.Nodes, cloneNode(node))
		}
	}
	return s.persistLocked()
}

// DeleteNode removes a single node by ID.
func (s *Store) DeleteNode(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, n := range s.state.Nodes {
		if n.ID == id {
			s.state.Nodes = append(s.state.Nodes[:i], s.state.Nodes[i+1:]...)
			return s.persistLocked()
		}
	}
	return ErrNotFound
}

// ListProfiles returns copies of all routing profiles.
func (s *Store) ListProfiles() []api.RoutingProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]api.RoutingProfile, len(s.state.Profiles))
	for i, p := range s.state.Profiles {
		out[i] = cloneProfile(p)
	}
	return out
}

// CreateProfile creates a new routing profile with strictly zero hidden default rules.
func (s *Store) CreateProfile(id, nameRu, nameEn, defaultTarget string) (api.RoutingProfile, error) {
	if id == "" {
		return api.RoutingProfile{}, errors.New("storage: profile ID must not be empty")
	}
	if defaultTarget == "" {
		defaultTarget = "direct"
	}
	prof := api.RoutingProfile{
		ID:            id,
		NameRu:        nameRu,
		NameEn:        nameEn,
		DefaultTarget: defaultTarget,
		Rules:         []api.RoutingRule{},
	}
	if err := s.UpsertProfile(prof); err != nil {
		return api.RoutingProfile{}, err
	}
	return prof, nil
}

// UpsertProfile inserts or updates a routing profile.
func (s *Store) UpsertProfile(prof api.RoutingProfile) error {
	if prof.ID == "" {
		return errors.New("storage: profile ID must not be empty")
	}
	if prof.Rules == nil {
		prof.Rules = []api.RoutingRule{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.state.Profiles {
		if existing.ID == prof.ID {
			s.state.Profiles[i] = cloneProfile(prof)
			return s.persistLocked()
		}
	}
	s.state.Profiles = append(s.state.Profiles, cloneProfile(prof))
	return s.persistLocked()
}

// DeleteProfile removes a routing profile by ID.
func (s *Store) DeleteProfile(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, p := range s.state.Profiles {
		if p.ID == id {
			s.state.Profiles = append(s.state.Profiles[:i], s.state.Profiles[i+1:]...)
			return s.persistLocked()
		}
	}
	return ErrNotFound
}
