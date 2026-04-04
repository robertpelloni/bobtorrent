package tui

// ──────────────────────────────────────────────────────────────────────────────
// Bobtorrent Supernode Terminal UI (TUI)
// ──────────────────────────────────────────────────────────────────────────────
// A cyberpunk-themed real-time dashboard built on Bubble Tea (github.com/
// charmbracelet/bubbletea) and Lip Gloss. Provides visibility into:
//   - Node wallet address and BOB balance
//   - Live lattice block feed (color-coded by block type)
//   - Storage market bid table
//   - Network statistics (peers, torrents, bandwidth)
//   - System health indicators
//
// The TUI receives data via Bubble Tea messages from the Supernode's
// background goroutines (market poller, torrent engine, P2P layer).
// ──────────────────────────────────────────────────────────────────────────────

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ════════════════════════════════════════════════════════════════════════════
// Styles
// ════════════════════════════════════════════════════════════════════════════

var (
	// titleStyle renders the main header in bright green (Matrix-style).
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00FF00")).
			Background(lipgloss.Color("#000000")).
			Padding(0, 2)

	// subtitleStyle renders section headers in cyan.
	subtitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00FFFF"))

	// statusOnline renders "ONLINE" indicators in green.
	statusOnline = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Bold(true)

	// statusOffline renders "OFFLINE" indicators in red.
	statusOffline = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	// blockSend colors send blocks in orange.
	blockSend = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8C00"))

	// blockReceive colors receive blocks in green.
	blockReceive = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))

	// blockMarket colors market blocks in yellow.
	blockMarket = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700"))

	// blockGovernance colors governance blocks in purple.
	blockGovernance = lipgloss.NewStyle().Foreground(lipgloss.Color("#9B59B6"))

	// blockNFT colors NFT blocks in magenta.
	blockNFT = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00FF"))

	// blockSwap colors swap blocks in teal.
	blockSwap = lipgloss.NewStyle().Foreground(lipgloss.Color("#1ABC9C"))

	// blockDefault colors other blocks in white.
	blockDefault = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))

	// tableStyle wraps the bid table in a subtle border.
	tableStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))

	// footerStyle renders the bottom help text in dim gray.
	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	// balanceStyle renders the balance in bright yellow.
	balanceStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD700")).
			Bold(true)

	// statsStyle renders network statistics.
	statsStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#87CEEB"))
)

// ════════════════════════════════════════════════════════════════════════════
// Message Types
// ════════════════════════════════════════════════════════════════════════════

// StatusMsg updates the status line and balance display.
type StatusMsg struct {
	Text    string
	Balance int64
}

// BidsMsg updates the storage market bid table.
type BidsMsg struct {
	Bids []Bid
}

// BlockFeedMsg adds a new block to the live block feed.
type BlockFeedMsg struct {
	Type      string // Block type (send, receive, mint_nft, etc.)
	Hash      string // Block hash (truncated for display)
	Account   string // Account that created the block
	Amount    int64  // Balance change (if applicable)
	Timestamp time.Time
}

// NetworkStatsMsg updates network-level statistics.
type NetworkStatsMsg struct {
	Peers       int
	Torrents    int
	Chains      int
	TotalBlocks int
	WSClients   int
}

// Bid represents a storage market bid for table display.
type Bid struct {
	ID     string `json:"id"`
	Magnet string `json:"magnet"`
	Amount int64  `json:"amount"`
	Status string `json:"status"`
}

// ════════════════════════════════════════════════════════════════════════════
// Model
// ════════════════════════════════════════════════════════════════════════════

// maxFeedLines limits the block feed to the most recent N entries.
const maxFeedLines = 15

// Model is the Bubble Tea model for the Supernode TUI dashboard.
type Model struct {
	// table displays the current storage market bids.
	table table.Model

	// status is the current node status text.
	status string

	// balance is the node wallet's current BOB balance.
	balance int64

	// lastPoll is the timestamp of the last market poll.
	lastPoll time.Time

	// blockFeed stores the most recent block feed entries for display.
	blockFeed []BlockFeedMsg

	// stats holds the latest network statistics snapshot.
	stats NetworkStatsMsg

	// latticeOnline tracks whether the lattice API is reachable.
	latticeOnline bool

	// width and height track terminal dimensions for responsive layout.
	width  int
	height int
}

// NewModel creates a new TUI Model with default state and table configuration.
func NewModel() Model {
	columns := []table.Column{
		{Title: "Bid ID", Width: 12},
		{Title: "Magnet URI", Width: 32},
		{Title: "Bounty (BOB)", Width: 14},
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
		Bold(true).
		Foreground(lipgloss.Color("#00FFFF"))
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return Model{
		table:         t,
		status:        "Initializing...",
		balance:       0,
		lastPoll:      time.Now(),
		blockFeed:     make([]BlockFeedMsg, 0),
		latticeOnline: false,
	}
}

// Init is called once at startup. No initial command needed.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles incoming messages and key events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case StatusMsg:
		m.status = msg.Text
		m.balance = msg.Balance
		m.lastPoll = time.Now()
		m.latticeOnline = msg.Text != "Lattice API Offline"

	case BidsMsg:
		var rows []table.Row
		for _, b := range msg.Bids {
			magnetDisplay := b.Magnet
			if len(magnetDisplay) > 29 {
				magnetDisplay = magnetDisplay[:29] + "..."
			}
			idDisplay := b.ID
			if len(idDisplay) > 10 {
				idDisplay = idDisplay[:10] + ".."
			}
			rows = append(rows, table.Row{
				idDisplay,
				magnetDisplay,
				fmt.Sprintf("%d", b.Amount),
				b.Status,
			})
		}
		m.table.SetRows(rows)
		m.latticeOnline = true

	case BlockFeedMsg:
		m.blockFeed = append(m.blockFeed, msg)
		if len(m.blockFeed) > maxFeedLines {
			m.blockFeed = m.blockFeed[len(m.blockFeed)-maxFeedLines:]
		}

	case NetworkStatsMsg:
		m.stats = msg
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// View renders the complete TUI dashboard.
func (m Model) View() string {
	var sb strings.Builder

	// ── Header ──
	sb.WriteString(titleStyle.Render("⚡ BOBTORRENT SUPERNODE v2.0 (GO) ⚡"))
	sb.WriteString("\n\n")

	// ── Status Bar ──
	latticeIndicator := statusOffline.Render("● OFFLINE")
	if m.latticeOnline {
		latticeIndicator = statusOnline.Render("● ONLINE")
	}

	statusLine := fmt.Sprintf("  Lattice: %s  │  Balance: %s  │  Status: %s  │  Poll: %s",
		latticeIndicator,
		balanceStyle.Render(fmt.Sprintf("%d BOB", m.balance)),
		m.status,
		m.lastPoll.Format("15:04:05"),
	)
	sb.WriteString(statusLine)
	sb.WriteString("\n\n")

	// ── Network Stats ──
	if m.stats.Peers > 0 || m.stats.Chains > 0 {
		statsLine := statsStyle.Render(fmt.Sprintf(
			"  Peers: %d  │  Chains: %d  │  Blocks: %d  │  Torrents: %d  │  WS Clients: %d",
			m.stats.Peers, m.stats.Chains, m.stats.TotalBlocks, m.stats.Torrents, m.stats.WSClients,
		))
		sb.WriteString(statsLine)
		sb.WriteString("\n\n")
	}

	// ── Live Block Feed ──
	sb.WriteString(subtitleStyle.Render("  ◈ LIVE BLOCK FEED"))
	sb.WriteString("\n")

	if len(m.blockFeed) == 0 {
		sb.WriteString("    Waiting for blocks...\n")
	} else {
		for _, entry := range m.blockFeed {
			line := renderBlockFeedLine(entry)
			sb.WriteString("    ")
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}
	sb.WriteString("\n")

	// ── Market Bids Table ──
	sb.WriteString(subtitleStyle.Render("  ◈ STORAGE MARKET BIDS"))
	sb.WriteString("\n")
	sb.WriteString(tableStyle.Render(m.table.View()))
	sb.WriteString("\n\n")

	// ── Footer ──
	sb.WriteString(footerStyle.Render("  Press 'q' to quit  │  Bobtorrent Omni-Workspace v11.6.0"))
	sb.WriteString("\n")

	return sb.String()
}

// renderBlockFeedLine formats a single block feed entry with color-coding
// based on the block type, matching the Node.js frontend's Dashboard feed.
func renderBlockFeedLine(entry BlockFeedMsg) string {
	ts := entry.Timestamp.Format("15:04:05")
	hashShort := entry.Hash
	if len(hashShort) > 12 {
		hashShort = hashShort[:12]
	}
	accountShort := entry.Account
	if len(accountShort) > 10 {
		accountShort = accountShort[:10]
	}

	label := fmt.Sprintf("[%s] %s %s..%s", ts, strings.ToUpper(entry.Type), accountShort, hashShort)

	if entry.Amount != 0 {
		label += fmt.Sprintf(" (%d BOB)", entry.Amount)
	}

	switch entry.Type {
	case "send":
		return blockSend.Render(label)
	case "receive", "open":
		return blockReceive.Render(label)
	case "market_bid", "accept_bid":
		return blockMarket.Render(label)
	case "proposal", "vote":
		return blockGovernance.Render(label)
	case "mint_nft", "transfer_nft":
		return blockNFT.Render(label)
	case "initiate_swap", "claim_swap", "refund_swap":
		return blockSwap.Render(label)
	default:
		return blockDefault.Render(label)
	}
}
