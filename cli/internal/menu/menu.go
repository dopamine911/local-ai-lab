// Package menu is the interactive launcher shown by `lab` with no arguments on a terminal (Bubble Tea v2).
// It lists the commands; Enter quits the menu and the chosen command runs normally in the terminal.
package menu

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Item is one command in the menu.
type Item struct {
	Name, Short string
}

type model struct {
	items  []Item
	cursor int
	chosen string
}

var (
	title  = lipgloss.NewStyle().Bold(true)
	sel    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	dim    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	nameW  = 0
)

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter", "space":
			m.chosen = m.items[m.cursor].Name
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() tea.View {
	var b strings.Builder
	b.WriteString(title.Render("lab: the homelab's commands") + "\n\n")
	for i, it := range m.items {
		line := fmt.Sprintf("%-*s  %s", nameW, it.Name, it.Short)
		if i == m.cursor {
			b.WriteString(sel.Render("> "+line) + "\n")
		} else {
			b.WriteString("  " + line + "\n")
		}
	}
	b.WriteString("\n" + dim.Render("↑/↓ choose · enter run · q quit · `lab <command> --help` for details") + "\n")
	v := tea.NewView(b.String())
	v.WindowTitle = "lab"
	return v
}

// Run shows the menu and returns the chosen command name ("" if the user quit).
func Run(items []Item) (string, error) {
	for _, it := range items {
		if len(it.Name) > nameW {
			nameW = len(it.Name)
		}
	}
	final, err := tea.NewProgram(model{items: items}).Run()
	if err != nil {
		return "", err
	}
	return final.(model).chosen, nil
}
