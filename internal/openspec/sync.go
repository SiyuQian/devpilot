package openspec

// SyncTarget abstracts the task board (Trello or GitHub Issues).
type SyncTarget interface {
	FindByName(name string) (id string, err error) // returns "" if not found
	Create(name, desc string) error
	Update(id, desc string) error
}

// SyncResult describes what happened to each change during sync.
type SyncResult struct {
	Name   string // change name
	Action string // "created" or "updated"
}

// Sync creates or updates board items for each change.
func Sync(changes []Change, target SyncTarget) ([]SyncResult, error) {
	var results []SyncResult
	for _, ch := range changes {
		existingID, err := target.FindByName(ch.Name)
		if err != nil {
			return results, err
		}
		if existingID != "" {
			if err := target.Update(existingID, ch.Description); err != nil {
				return results, err
			}
			results = append(results, SyncResult{Name: ch.Name, Action: "updated"})
		} else {
			if err := target.Create(ch.Name, ch.Description); err != nil {
				return results, err
			}
			results = append(results, SyncResult{Name: ch.Name, Action: "created"})
		}
	}
	return results, nil
}
