package skillmgr

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// SelectSkillsFromCatalog presents an interactive multi-select checklist and
// returns the names of the selected skills. Returns nil if the user cancels.
func SelectSkillsFromCatalog(catalog []CatalogEntry) ([]string, error) {
	m := newMultiSelectModel(catalog)
	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return nil, err
	}
	final := result.(multiSelectModel)
	if final.cancelled {
		return nil, nil
	}
	var selected []string
	for _, item := range final.items {
		if item.checked {
			selected = append(selected, item.name)
		}
	}
	return selected, nil
}

type multiSelectItem struct {
	name    string
	desc    string
	checked bool
}

type multiSelectModel struct {
	items     []multiSelectItem
	cursor    int
	cancelled bool
	done      bool
}

func newMultiSelectModel(catalog []CatalogEntry) multiSelectModel {
	items := make([]multiSelectItem, len(catalog))
	for i, e := range catalog {
		items[i] = multiSelectItem{name: e.Name, desc: e.Description}
	}
	return multiSelectModel{items: items}
}

func (m multiSelectModel) Init() tea.Cmd {
	return nil
}

func (m multiSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.cancelled = true
			return m, tea.Quit
		case "enter":
			m.done = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case " ":
			m.items[m.cursor].checked = !m.items[m.cursor].checked
		case "a":
			// Toggle all: if any unchecked, check all; otherwise uncheck all
			anyUnchecked := false
			for _, item := range m.items {
				if !item.checked {
					anyUnchecked = true
					break
				}
			}
			for i := range m.items {
				m.items[i].checked = anyUnchecked
			}
		}
	}
	return m, nil
}

func (m multiSelectModel) View() string {
	if m.done || m.cancelled {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("  Select skills to install (space to toggle, a to toggle all, enter to confirm):\n\n")

	for i, item := range m.items {
		cursor := "  "
		if m.cursor == i {
			cursor = "> "
		}
		check := "[ ]"
		if item.checked {
			check = "[x]"
		}
		fmt.Fprintf(&sb, "  %s%s %-30s %s\n", cursor, check, item.name, item.desc)
	}
	sb.WriteString("\n  (q to cancel)\n")
	return sb.String()
}
