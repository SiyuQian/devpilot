// Package trello provides a minimal Trello REST API client and the
// `devpilot push` command for creating cards from plan files.
package trello

// Board represents a Trello board.
type Board struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// List represents a Trello list on a board.
type List struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Label represents a Trello card label.
type Label struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// Card represents a Trello card.
type Card struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Desc     string  `json:"desc"`
	IDList   string  `json:"idList"`
	ShortURL string  `json:"shortUrl"`
	Labels   []Label `json:"labels"`
}
