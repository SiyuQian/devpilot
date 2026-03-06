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

func (t *TrelloTarget) Create(name, desc string) error {
	_, err := t.client.CreateCard(t.listID, name, desc)
	return err
}

func (t *TrelloTarget) Update(id, desc string) error {
	return t.client.UpdateCard(id, desc)
}
