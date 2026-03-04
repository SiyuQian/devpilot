package taskrunner

import (
	"fmt"

	"github.com/siyuqian/devpilot/internal/trello"
)

// TrelloSource implements TaskSource for Trello boards.
type TrelloSource struct {
	client       *trello.Client
	boardName    string
	readyListID  string
	inProgListID string
	doneListID   string
	failedListID string
}

func NewTrelloSource(client *trello.Client, boardName string) *TrelloSource {
	return &TrelloSource{client: client, boardName: boardName}
}

func (s *TrelloSource) Init() (SourceInfo, error) {
	board, err := s.client.FindBoardByName(s.boardName)
	if err != nil {
		return SourceInfo{}, fmt.Errorf("find board: %w", err)
	}

	listNames := map[string]*string{
		"Ready":       &s.readyListID,
		"In Progress": &s.inProgListID,
		"Done":        &s.doneListID,
		"Failed":      &s.failedListID,
	}
	resolved := make(map[string]string, len(listNames))
	for name, idPtr := range listNames {
		list, err := s.client.FindListByName(board.ID, name)
		if err != nil {
			return SourceInfo{}, fmt.Errorf("find list %q: %w", name, err)
		}
		*idPtr = list.ID
		resolved[name] = list.ID
	}
	return SourceInfo{
		DisplayName: board.Name,
		BoardID:     board.ID,
		Lists:       resolved,
	}, nil
}

func (s *TrelloSource) FetchReady() ([]Task, error) {
	cards, err := s.client.GetListCards(s.readyListID)
	if err != nil {
		return nil, err
	}
	tasks := make([]Task, 0, len(cards))
	for _, c := range cards {
		tasks = append(tasks, Task{
			ID:          c.ID,
			Name:        c.Name,
			Description: c.Desc,
			URL:         c.ShortURL,
			Priority:    trelloPriority(c),
		})
	}
	return tasks, nil
}

func (s *TrelloSource) MarkInProgress(id string) error {
	return s.client.MoveCard(id, s.inProgListID)
}

func (s *TrelloSource) MarkDone(id, comment string) error {
	if err := s.client.MoveCard(id, s.doneListID); err != nil {
		return err
	}
	return s.client.AddComment(id, comment)
}

func (s *TrelloSource) MarkFailed(id, comment string) error {
	if err := s.client.MoveCard(id, s.failedListID); err != nil {
		return err
	}
	return s.client.AddComment(id, comment)
}

func trelloPriority(c trello.Card) int {
	names := make([]string, len(c.Labels))
	for i, l := range c.Labels {
		names[i] = l.Name
	}
	return priorityFromLabelNames(names)
}
