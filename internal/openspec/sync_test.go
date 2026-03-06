package openspec

import "testing"

type mockTarget struct {
	created []struct{ name, desc string }
	updated []struct{ id, desc string }
	cards   map[string]string // name -> id
}

func (m *mockTarget) FindByName(name string) (string, error) {
	if id, ok := m.cards[name]; ok {
		return id, nil
	}
	return "", nil
}

func (m *mockTarget) Create(name, desc string) error {
	m.created = append(m.created, struct{ name, desc string }{name, desc})
	return nil
}

func (m *mockTarget) Update(id, desc string) error {
	m.updated = append(m.updated, struct{ id, desc string }{id, desc})
	return nil
}

func TestSync_createsNew(t *testing.T) {
	mock := &mockTarget{cards: map[string]string{}}
	changes := []Change{
		{Name: "add-auth", Description: "implement auth"},
	}

	results, err := Sync(changes, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Action != "created" {
		t.Errorf("expected action=created, got %s", results[0].Action)
	}
	if results[0].Name != "add-auth" {
		t.Errorf("expected name=add-auth, got %s", results[0].Name)
	}
	if len(mock.created) != 1 {
		t.Fatalf("expected 1 create call, got %d", len(mock.created))
	}
	if mock.created[0].name != "add-auth" || mock.created[0].desc != "implement auth" {
		t.Errorf("unexpected create args: %+v", mock.created[0])
	}
	if len(mock.updated) != 0 {
		t.Errorf("expected no update calls, got %d", len(mock.updated))
	}
}

func TestSync_updatesExisting(t *testing.T) {
	mock := &mockTarget{
		cards: map[string]string{"add-auth": "card123"},
	}
	changes := []Change{
		{Name: "add-auth", Description: "updated plan"},
	}

	results, err := Sync(changes, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Action != "updated" {
		t.Errorf("expected action=updated, got %s", results[0].Action)
	}
	if results[0].Name != "add-auth" {
		t.Errorf("expected name=add-auth, got %s", results[0].Name)
	}
	if len(mock.updated) != 1 {
		t.Fatalf("expected 1 update call, got %d", len(mock.updated))
	}
	if mock.updated[0].id != "card123" || mock.updated[0].desc != "updated plan" {
		t.Errorf("unexpected update args: %+v", mock.updated[0])
	}
	if len(mock.created) != 0 {
		t.Errorf("expected no create calls, got %d", len(mock.created))
	}
}
