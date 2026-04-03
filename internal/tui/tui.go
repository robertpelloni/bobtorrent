package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type Model struct {
	table    table.Model
	status   string
	balance  int64
	lastPoll time.Time
}

func NewModel() Model {
	columns := []table.Column{
		{Title: "Bid ID", Width: 10},
		{Title: "Magnet", Width: 30},
		{Title: "Amount", Width: 10},
		{Title: "Status", Width: 10},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(7),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return Model{
		table:    t,
		status:   "Initializing...",
		balance:  0,
		lastPoll: time.Now(),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case StatusMsg:
		m.status = msg.Text
		m.balance = msg.Balance
		m.lastPoll = time.Now()
	case BidsMsg:
		var rows []table.Row
		for _, b := range msg.Bids {
			rows = append(rows, table.Row{b.ID[:8], b.Magnet[:27] + "...", fmt.Sprintf("%d", b.Amount), b.Status})
		}
		m.table.SetRows(rows)
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FF00")).
		Render("BOBTORRENT SUPERNODE (GO)")

	statusLine := fmt.Sprintf("Status: %s | Balance: %d BOB | Last Poll: %s", 
		m.status, m.balance, m.lastPoll.Format("15:04:05"))

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"\n",
		statusLine,
		"\n",
		baseStyle.Render(m.table.View()),
		"\nPress 'q' to quit",
	)
}

type StatusMsg struct {
	Text    string
	Balance int64
}

type BidsMsg struct {
	Bids []Bid
}

type Bid struct {
	ID     string
	Magnet string
	Amount int64
	Status string
}
