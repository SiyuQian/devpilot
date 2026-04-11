package openspec

import "github.com/siyuqian/devpilot/internal/trello"

// TrelloTarget implements SyncTarget by delegating to a trello.Client.
type TrelloTarget struct {
	client *trello.Client
	listID string
}

// NewTrelloTarget creates a TrelloTarget that syncs to the given Trello list.
func NewTrelloTarget(client *trello.Client, listID string) *TrelloTarget {
	return &TrelloTarget{client: client, listID: listID}
}

// FindByName returns the ID of the card with the given name in the configured
// list, or an empty string if no such card exists.
func (t *TrelloTarget) FindByName(name string) (string, error) {
	card, err := t.client.FindCardByName(t.listID, name)
	if err != nil {
		return "", err
	}
	if card == nil {
		return "", nil
	}
	return card.ID, nil
}

// Create adds a new card with the given name and description to the
// configured Trello list.
func (t *TrelloTarget) Create(name, desc string) error {
	_, err := t.client.CreateCard(t.listID, name, desc)
	return err
}

// Update replaces the description of the card with the given ID.
func (t *TrelloTarget) Update(id, desc string) error {
	return t.client.UpdateCard(id, desc)
}
