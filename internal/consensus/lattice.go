package consensus

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"bobtorrent/pkg/torrent"
)

const (
	DemurrageRatePerMS = 0.0001 / 60000
	ProposalCost       = 10
	NFTMintCost        = 50
	NFTTransferFee     = 1
	Epsilon            = 0.001
)

type Lattice struct {
	chains     map[string][]*torrent.Block
	blocks     map[string]*torrent.Block
	pending    map[string][]PendingTx
	proposals  map[string]*Proposal
	votes      map[string]map[string]Vote
	marketBids map[string]*MarketBid
	swaps      map[string]*Swap
	nfts       map[string]*NFT
	stateHash  string
	mu         sync.RWMutex
}

type PendingTx struct {
	Hash    string      `json:"hash"`
	Amount  int64       `json:"amount"`
	Sender  string      `json:"sender"`
	Payload interface{} `json:"payload"`
}

type Proposal struct {
	ID           string    `json:"id"`
	Proposer     string    `json:"proposer"`
	Title        string    `json:"title"`
	Status       string    `json:"status"`
	VotesFor     float64   `json:"votesFor"`
	VotesAgainst float64   `json:"votesAgainst"`
	EndTime      string    `json:"endTime"`
	Timestamp    int64     `json:"timestamp"`
}

type Vote struct {
	Type  string  `json:"type"`
	Power float64 `json:"power"`
}

type MarketBid struct {
	ID         string `json:"id"`
	Creator    string `json:"creator"`
	Magnet     string `json:"magnet"`
	Amount     int64  `json:"amount"`
	Status     string `json:"status"`
	AcceptedBy string `json:"acceptedBy,omitempty"`
	Timestamp  int64  `json:"timestamp"`
}

type Swap struct {
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
	Amount    int64  `json:"amount"`
	Expiry    int64  `json:"expiry"`
	Status    string `json:"status"`
	Claimer   string `json:"claimer,omitempty"`
}

type NFT struct {
	ID          string `json:"id"`
	Owner       string `json:"owner"`
	Name        string `json:"name"`
	Magnet      string `json:"magnet"`
	Description string `json:"description"`
	Timestamp   int64  `json:"timestamp"`
}

func NewLattice() *Lattice {
	return &Lattice{
		chains:     make(map[string][]*torrent.Block),
		blocks:     make(map[string]*torrent.Block),
		pending:    make(map[string][]PendingTx),
		proposals:  make(map[string]*Proposal),
		votes:      make(map[string]map[string]Vote),
		marketBids: make(map[string]*MarketBid),
		swaps:      make(map[string]*Swap),
		nfts:       make(map[string]*NFT),
		stateHash:  strings.Repeat("0", 64),
	}
}

func (l *Lattice) GetFrontier(account string) *torrent.Block {
	chain := l.chains[account]
	if len(chain) == 0 {
		return nil
	}
	return chain[len(chain)-1]
}

func (l *Lattice) GetBalance(account string) int64 {
	f := l.GetFrontier(account)
	if f == nil {
		return 0
	}
	return l.ApplyDemurrage(f.Balance, f.Timestamp, time.Now().UnixMilli())
}

func (l *Lattice) ApplyDemurrage(balance int64, lastTs, currentTs int64) int64 {
	if lastTs == 0 || balance <= 0 {
		return balance
	}
	elapsed := currentTs - lastTs
	if elapsed <= 0 {
		return balance
	}
	decay := float64(balance) * DemurrageRatePerMS * float64(elapsed)
	res := float64(balance) - decay
	if res < 0 {
		return 0
	}
	return int64(math.Round(res))
}

func (l *Lattice) ProcessBlock(b *torrent.Block) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 1. Verify Signature (Mocked in torrent package if not implemented fully)
	// In production, we'd verify Base58 Ed25519 signature
	
	// 2. Chain Integrity
	frontier := l.GetFrontier(b.Account)
	if b.Type == "open" {
		if frontier != nil {
			return fmt.Errorf("account already open")
		}
		if b.Previous != nil {
			return fmt.Errorf("open block must have no previous")
		}
	} else {
		if frontier == nil {
			return fmt.Errorf("account not open")
		}
		if b.Previous == nil || *b.Previous != frontier.Hash {
			return fmt.Errorf("invalid previous block hash")
		}
		if b.Height != frontier.Height+1 {
			return fmt.Errorf("invalid block height")
		}
	}

	// 3. SPoRA Verification (Simplified for Go Port)
	// ... (logic similar to Node.js)

	// 4. State Transitions
	prevBalance := int64(0)
	if frontier != nil {
		prevBalance = l.ApplyDemurrage(frontier.Balance, frontier.Timestamp, b.Timestamp)
	}

	switch b.Type {
	case "send":
		amount := prevBalance - b.Balance
		if amount <= 0 {
			return fmt.Errorf("send block must decrease balance")
		}
		l.pending[b.Link] = append(l.pending[b.Link], PendingTx{
			Hash:    b.Hash,
			Amount:  amount,
			Sender:  b.Account,
			Payload: b.Payload,
		})

	case "receive":
		pending := l.pending[b.Account]
		found := false
		var tx PendingTx
		for i, p := range pending {
			if p.Hash == b.Link {
				tx = p
				l.pending[b.Account] = append(pending[:i], pending[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("pending transaction not found")
		}
		if b.Balance != prevBalance+tx.Amount {
			return fmt.Errorf("invalid receive balance")
		}

	case "market_bid":
		amount := prevBalance - b.Balance
		if amount <= 0 {
			return fmt.Errorf("market bid must cost BOB")
		}
		magnet := ""
		if m, ok := b.Payload.(map[string]interface{})["magnet"].(string); ok {
			magnet = m
		}
		l.marketBids[b.Hash] = &MarketBid{
			ID:        b.Hash,
			Creator:   b.Account,
			Magnet:    magnet,
			Amount:    amount,
			Status:    "OPEN",
			Timestamp: b.Timestamp,
		}

	case "accept_bid":
		bid := l.marketBids[b.Link]
		if bid == nil || bid.Status != "OPEN" {
			return fmt.Errorf("bid not found or already accepted")
		}
		if b.Balance != prevBalance+bid.Amount {
			return fmt.Errorf("invalid accept_bid balance")
		}
		bid.Status = "ACCEPTED"
		bid.AcceptedBy = b.Account

	case "open":
		// Handle initial balance (e.g. from System Genesis)
		if b.Link == "SYSTEM_GENESIS" && len(l.chains) == 0 {
			// Genesis block allowed
		} else {
			// Standard open must receive a pending transaction
			// ... logic similar to receive
		}
	}

	// 5. Finalize
	l.chains[b.Account] = append(l.chains[b.Account], b)
	l.blocks[b.Hash] = b
	l.updateStateHash(b)

	return nil
}

func (l *Lattice) updateStateHash(b *torrent.Block) {
	raw := l.stateHash + b.Hash
	hash := sha256.Sum256([]byte(raw))
	l.stateHash = hex.EncodeToString(hash[:])
}
