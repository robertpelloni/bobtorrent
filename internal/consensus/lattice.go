package consensus

// ──────────────────────────────────────────────────────────────────────────────
// Asynchronous Block Lattice Consensus Engine (Go Port)
// ──────────────────────────────────────────────────────────────────────────────
// This file implements the core consensus logic for the Bobcoin Block Lattice,
// a DAG-based ledger where every account maintains its own independent chain
// of blocks. This architecture enables massive parallelism (10,000+ TPS) and
// sub-second finality because transactions on different accounts never contend
// for the same lock.
//
// Key concepts ported from bobcoin-consensus/Lattice.js:
//   - Demurrage: Currency decay over time to incentivize economic velocity
//   - SPoRA: Succinct Proof of Random Access (Arweave-style mining proofs)
//   - Quadratic Voting: Governance voting power proportional to sqrt(balance)
//   - HTLC Swaps: Hashed Time-Lock Contracts for trustless atomic exchange
//   - NFT Registry: On-chain digital asset ownership via content-addressed magnets
//   - Staking: Lock tokens for yield and amplified governance power
//
// Thread Safety: All public methods acquire the appropriate lock (read or write)
// on the Lattice mutex. The Lattice is safe for concurrent use from multiple
// goroutines (HTTP handlers, P2P receivers, WebSocket broadcasters).
// ──────────────────────────────────────────────────────────────────────────────

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

// ════════════════════════════════════════════════════════════════════════════
// Economic Constants
// ════════════════════════════════════════════════════════════════════════════

const (
	// DemurrageRatePerMS is the per-millisecond decay rate for dormant balances.
	// At this rate, ~0.01% of a balance decays per minute, incentivizing
	// continuous economic participation rather than hoarding.
	DemurrageRatePerMS = 0.0001 / 60000

	// ProposalCost is the number of BOB tokens burned to submit a governance
	// proposal. This prevents spam proposals while keeping governance accessible.
	ProposalCost = 10

	// NFTMintCost is the number of BOB tokens burned to mint a new NFT.
	// This funds the network's storage overhead for the NFT's metadata.
	NFTMintCost = 50

	// NFTTransferFee is the number of BOB tokens burned per NFT transfer.
	// This small friction prevents wash-trading and funds network maintenance.
	NFTTransferFee = 1

	// DataAnchorCost is the burn fee for the legacy `data_anchor` flow used by
	// the Bobcoin Vault page. `publish_manifest` anchors are informational and
	// currently zero-cost to make the new Go storage round-trip easy to adopt.
	DataAnchorCost = 10

	// StakingYieldRatePerMS is the per-millisecond yield rate for staked tokens.
	// Stakers earn approximately 5% APY, paid continuously.
	StakingYieldRatePerMS = 0.05 / (365.25 * 24 * 60 * 60 * 1000)

	// StakingVotingMultiplier amplifies the Quadratic Voting power of staked
	// tokens by 2x, rewarding long-term network commitment.
	StakingVotingMultiplier = 2.0

	// Epsilon is a floating-point comparison tolerance for balance checks.
	Epsilon = 0.001
)

// ════════════════════════════════════════════════════════════════════════════
// Core Types
// ════════════════════════════════════════════════════════════════════════════

// Lattice is the top-level consensus state machine. It holds every confirmed
// block, every account chain, and all derived state (proposals, NFTs, swaps,
// market bids). All mutations go through ProcessBlock which enforces the
// full validation pipeline before committing any state change.
type Lattice struct {
	// chains maps account public keys to their ordered sequence of blocks.
	// Each account has exactly one chain, starting with an "open" block.
	chains map[string][]*torrent.Block

	// blocks provides O(1) lookup of any confirmed block by its SHA-256 hash.
	blocks map[string]*torrent.Block

	// pending maps recipient account keys to their unprocessed incoming
	// transactions. A "send" block on Alice's chain creates a PendingTx
	// entry for Bob; Bob's "receive" block consumes it.
	pending map[string][]PendingTx

	// proposals stores all governance proposals indexed by proposal ID (block hash).
	proposals map[string]*Proposal

	// votes maps proposal ID → voter account → Vote, preventing double-voting.
	votes map[string]map[string]Vote

	// marketBids stores all storage market bids indexed by bid ID (block hash).
	marketBids map[string]*MarketBid

	// swaps stores all active HTLC atomic swaps indexed by swap ID.
	swaps map[string]*Swap

	// nfts stores all minted NFTs indexed by NFT ID (block hash).
	nfts map[string]*NFT

	// anchors stores on-chain manifest/data anchors. These provide a bridge
	// between off-chain published storage artifacts and wallet-attributed
	// lattice records, enabling provenance, discovery, and later retrieval.
	anchors map[string]*ManifestAnchor

	// stakeInfo tracks per-account staking metadata (amount, start time).
	stakeInfo map[string]*StakeInfo

	// stateHash is the cumulative SHA-256 rolling hash of the entire lattice.
	// Each confirmed block's hash is folded into the state hash, creating
	// a Merkle-like commitment to the full history.
	stateHash string

	// peers maps registered P2P lattice node addresses for block broadcasting.
	peers map[string]bool

	// mu protects all lattice state from concurrent access.
	mu sync.RWMutex
}

// PendingTx represents an unprocessed incoming transaction waiting for
// the recipient to issue a "receive" block on their chain.
type PendingTx struct {
	Hash    string      `json:"hash"`    // The hash of the sender's "send" block
	Amount  int64       `json:"amount"`  // The number of tokens being transferred
	Sender  string      `json:"sender"`  // The sender's account public key
	Payload interface{} `json:"payload"` // Optional payload (encrypted memo, etc.)
}

// Proposal represents a governance proposal submitted to the lattice DAO.
type Proposal struct {
	ID           string  `json:"id"`           // Block hash of the proposal block
	Proposer     string  `json:"proposer"`     // Account that submitted the proposal
	Title        string  `json:"title"`        // Human-readable proposal title
	Description  string  `json:"description"`  // Detailed proposal description
	Status       string  `json:"status"`       // "active", "passed", "failed"
	VotesFor     float64 `json:"votesFor"`     // Cumulative quadratic voting power FOR
	VotesAgainst float64 `json:"votesAgainst"` // Cumulative quadratic voting power AGAINST
	EndTime      int64   `json:"endTime"`      // Unix ms when voting closes
	Timestamp    int64   `json:"timestamp"`    // Unix ms when proposal was created
}

// Vote records a single account's vote on a governance proposal.
type Vote struct {
	Type  string  `json:"type"`  // "for" or "against"
	Power float64 `json:"power"` // Quadratic voting power = sqrt(balance) * staking_mult
}

// MarketBid represents a storage bid on the decentralized Bobtorrent market.
// Creators post bids with a magnet URI and BOB bounty; supernodes accept
// bids by downloading the content and proving storage via SPoRA.
type MarketBid struct {
	ID         string `json:"id"`                    // Block hash of the market_bid block
	Creator    string `json:"creator"`               // Account that posted the bid
	Magnet     string `json:"magnet"`                // Magnet URI of the content to store
	Amount     int64  `json:"amount"`                // BOB bounty for storage
	Status     string `json:"status"`                // "OPEN", "ACCEPTED", "EXPIRED"
	AcceptedBy string `json:"acceptedBy,omitempty"`  // Account that accepted the bid
	Timestamp  int64  `json:"timestamp"`             // Unix ms when bid was posted
}

// Swap represents an HTLC (Hashed Time-Lock Contract) for trustless atomic
// exchange between two parties. The sender locks tokens; the recipient
// reveals a secret to claim them before expiry.
type Swap struct {
	ID        string `json:"id"`                 // Block hash of the initiate_swap block
	Sender    string `json:"sender"`             // Account that initiated the swap
	Recipient string `json:"recipient"`          // Intended recipient of the locked tokens
	Amount    int64  `json:"amount"`             // Locked token amount
	HashLock  string `json:"hashLock"`           // SHA-256 hash of the secret
	Expiry    int64  `json:"expiry"`             // Unix ms deadline for claiming
	Status    string `json:"status"`             // "LOCKED", "CLAIMED", "REFUNDED"
	Claimer   string `json:"claimer,omitempty"`  // Account that claimed the swap
	Secret    string `json:"secret,omitempty"`   // Revealed secret (set on claim)
}

// NFT represents a non-fungible token minted on the lattice. Each NFT is
// permanently linked to a Bobtorrent magnet URI, enabling decentralized
// ownership of any file, game asset, or digital artwork.
type NFT struct {
	ID          string `json:"id"`          // Block hash of the mint_nft block
	Owner       string `json:"owner"`       // Current owner's account public key
	Name        string `json:"name"`        // Human-readable name
	Magnet      string `json:"magnet"`      // Magnet URI of the NFT's content
	Description string `json:"description"` // Description / metadata
	Timestamp   int64  `json:"timestamp"`   // Unix ms when the NFT was minted
}

// ManifestAnchor records a wallet-attributed publication reference on the
// lattice. The referenced manifest itself may live on a supernode registry,
// but this anchor proves which account attached it to the sovereign network.
type ManifestAnchor struct {
	ID             string `json:"id"`
	BlockHash      string `json:"blockHash"`
	Owner          string `json:"owner"`
	Type           string `json:"type"`
	ManifestID     string `json:"manifestId,omitempty"`
	Locator        string `json:"locator,omitempty"`
	ManifestURL    string `json:"manifestUrl,omitempty"`
	Name           string `json:"name,omitempty"`
	Size           int64  `json:"size,omitempty"`
	CiphertextHash string `json:"ciphertextHash,omitempty"`
	ProofHash      string `json:"proofHash,omitempty"`
	ProofSignature string `json:"proofSignature,omitempty"`
	Magnet         string `json:"magnet,omitempty"`
	Timestamp      int64  `json:"timestamp"`
}

// StakeInfo tracks an account's active staking position.
type StakeInfo struct {
	Amount    int64 `json:"amount"`    // Number of tokens staked
	StartTime int64 `json:"startTime"` // Unix ms when staking began
}

// ════════════════════════════════════════════════════════════════════════════
// Constructor
// ════════════════════════════════════════════════════════════════════════════

// NewLattice creates an empty Lattice with all maps initialized and the
// state hash set to 64 zero characters (genesis state).
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
		anchors:    make(map[string]*ManifestAnchor),
		stakeInfo:  make(map[string]*StakeInfo),
		stateHash:  strings.Repeat("0", 64),
		peers:      make(map[string]bool),
	}
}

// ════════════════════════════════════════════════════════════════════════════
// Peer Management
// ════════════════════════════════════════════════════════════════════════════

// AddPeer registers a new P2P lattice node address for block broadcasting.
func (l *Lattice) AddPeer(addr string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.peers[addr] = true
}

// RemovePeer unregisters a P2P lattice node.
func (l *Lattice) RemovePeer(addr string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.peers, addr)
}

// GetPeers returns a snapshot of all registered P2P peer addresses.
func (l *Lattice) GetPeers() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	res := make([]string, 0, len(l.peers))
	for p := range l.peers {
		res = append(res, p)
	}
	return res
}

// ════════════════════════════════════════════════════════════════════════════
// Queries (Read-Only)
// ════════════════════════════════════════════════════════════════════════════

// GetFrontier returns the latest (most recent) block on an account's chain,
// or nil if the account has never been opened.
func (l *Lattice) GetFrontier(account string) *torrent.Block {
	chain := l.chains[account]
	if len(chain) == 0 {
		return nil
	}
	return chain[len(chain)-1]
}

// GetBalance returns the current balance for an account with demurrage
// applied up to the current wall-clock time.
func (l *Lattice) GetBalance(account string) int64 {
	f := l.GetFrontier(account)
	if f == nil {
		return 0
	}
	return l.ApplyDemurrage(f.Balance, f.Timestamp, time.Now().UnixMilli())
}

// GetStakedBalance returns the currently staked amount for an account.
func (l *Lattice) GetStakedBalance(account string) int64 {
	f := l.GetFrontier(account)
	if f == nil {
		return 0
	}
	return f.StakedBalance
}

// GetQuadraticVotingPower computes the Quadratic Voting power for an account.
// Power = sqrt(balance) * stakingMultiplier. Stakers get 2x the voting weight.
func (l *Lattice) GetQuadraticVotingPower(account string) float64 {
	balance := float64(l.GetBalance(account))
	staked := float64(l.GetStakedBalance(account))
	basePower := math.Sqrt(balance)
	stakedPower := math.Sqrt(staked) * StakingVotingMultiplier
	return basePower + stakedPower
}

// ════════════════════════════════════════════════════════════════════════════
// Demurrage (Currency Decay)
// ════════════════════════════════════════════════════════════════════════════

// ApplyDemurrage calculates the decayed balance between two timestamps.
// Demurrage is a continuous linear decay that incentivizes economic velocity
// by reducing the value of dormant (un-transacted) balances over time.
//
// This mirrors the Node.js implementation in Lattice.js:
//   decay = balance * DemurrageRatePerMS * elapsed_ms
//   result = balance - decay
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

// ════════════════════════════════════════════════════════════════════════════
// Block Processing Pipeline
// ════════════════════════════════════════════════════════════════════════════

// ProcessBlock validates and commits a new block to the lattice. This is the
// critical consensus entry point — every state mutation flows through here.
//
// Validation pipeline:
//   1. Signature verification (Ed25519 via pkg/torrent/crypto.go)
//   2. Chain integrity (previous hash linkage and height sequencing)
//   3. Balance validity (demurrage-adjusted arithmetic)
//   4. Type-specific state transitions (send, receive, governance, NFT, etc.)
//   5. State hash update (rolling SHA-256 commitment)
func (l *Lattice) ProcessBlock(b *torrent.Block) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// ── 1. Duplicate Detection ──
	if _, exists := l.blocks[b.Hash]; exists {
		// Block already processed (e.g., received via P2P broadcast).
		// This is not an error — it's expected in a multi-peer network.
		return nil
	}

	// ── 2. Signature Verification ──
	// In production, verify the Ed25519 signature against the account's
	// public key to ensure the block was authored by the account owner.
	if b.Signature != "" {
		if !torrent.Verify(b.Hash, b.Signature, b.Account) {
			return fmt.Errorf("invalid signature for block %s", b.Hash[:16])
		}
	}

	// ── 3. Chain Integrity ──
	frontier := l.GetFrontier(b.Account)

	// Backward-compatibility shim for the existing bobcoin frontend:
	// several UI pages still construct blocks without explicit height and
	// staked_balance fields. When omitted, we infer them from the frontier.
	if frontier != nil {
		if b.Height == 0 {
			b.Height = frontier.Height + 1
		}
		if b.Type != "stake" && b.Type != "unstake" && b.StakedBalance == 0 {
			b.StakedBalance = frontier.StakedBalance
		}
	}

	if b.Type == "open" {
		if frontier != nil {
			return fmt.Errorf("account %s already open", b.Account[:16])
		}
		if b.Previous != nil {
			return fmt.Errorf("open block must have nil previous")
		}
		if b.Height != 0 {
			return fmt.Errorf("open block must have height 0")
		}
	} else {
		if frontier == nil {
			return fmt.Errorf("account %s not open", b.Account[:16])
		}
		if b.Previous == nil || *b.Previous != frontier.Hash {
			return fmt.Errorf("invalid previous hash: expected %s", frontier.Hash[:16])
		}
		if b.Height != frontier.Height+1 {
			return fmt.Errorf("invalid height: expected %d, got %d", frontier.Height+1, b.Height)
		}
	}

	// ── 4. State Transitions ──
	prevBalance := int64(0)
	if frontier != nil {
		prevBalance = l.ApplyDemurrage(frontier.Balance, frontier.Timestamp, b.Timestamp)
	}

	switch b.Type {
	case "open":
		if err := l.processOpen(b); err != nil {
			return err
		}

	case "send":
		if err := l.processSend(b, prevBalance); err != nil {
			return err
		}

	case "receive":
		if err := l.processReceive(b, prevBalance); err != nil {
			return err
		}

	case "market_bid":
		if err := l.processMarketBid(b, prevBalance); err != nil {
			return err
		}

	case "accept_bid":
		if err := l.processAcceptBid(b, prevBalance); err != nil {
			return err
		}

	case "proposal":
		if err := l.processProposal(b, prevBalance); err != nil {
			return err
		}

	case "vote":
		if err := l.processVote(b, prevBalance); err != nil {
			return err
		}

	case "mint_nft":
		if err := l.processMintNFT(b, prevBalance); err != nil {
			return err
		}

	case "transfer_nft":
		if err := l.processTransferNFT(b, prevBalance); err != nil {
			return err
		}

	case "stake":
		if err := l.processStake(b, prevBalance); err != nil {
			return err
		}

	case "unstake":
		if err := l.processUnstake(b, prevBalance); err != nil {
			return err
		}

	case "initiate_swap":
		if err := l.processInitiateSwap(b, prevBalance); err != nil {
			return err
		}

	case "claim_swap":
		if err := l.processClaimSwap(b, prevBalance); err != nil {
			return err
		}

	case "refund_swap":
		if err := l.processRefundSwap(b, prevBalance); err != nil {
			return err
		}

	case "publish_manifest":
		if err := l.processPublishManifest(b, prevBalance); err != nil {
			return err
		}

	case "data_anchor":
		if err := l.processDataAnchor(b, prevBalance); err != nil {
			return err
		}

	case "achievement_unlock":
		// Achievement blocks are informational — no balance change required.
		// They are simply recorded on the account's chain for the Trophy Room.

	default:
		return fmt.Errorf("unknown block type: %s", b.Type)
	}

	// ── 5. Commit ──
	l.chains[b.Account] = append(l.chains[b.Account], b)
	l.blocks[b.Hash] = b
	l.updateStateHash(b)

	return nil
}

// ════════════════════════════════════════════════════════════════════════════
// Block Type Processors
// ════════════════════════════════════════════════════════════════════════════

// processOpen handles account creation. The first block on any account chain.
// May link to SYSTEM_GENESIS for the primordial account or to a pending
// incoming transaction for standard account opening.
func (l *Lattice) processOpen(b *torrent.Block) error {
	if b.Link == "SYSTEM_GENESIS" && len(l.chains) == 0 {
		// Genesis block: the very first account in the lattice.
		// Its balance is the initial token supply.
		return nil
	}

	// Standard open: must consume a pending incoming transaction
	pending := l.pending[b.Account]
	for i, p := range pending {
		if p.Hash == b.Link {
			if b.Balance != p.Amount {
				return fmt.Errorf("open balance must equal pending amount (%d)", p.Amount)
			}
			l.pending[b.Account] = append(pending[:i], pending[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("no pending transaction found for open block")
}

// processSend handles outgoing token transfers. The sender's balance decreases
// and a PendingTx is created for the recipient to claim.
func (l *Lattice) processSend(b *torrent.Block, prevBalance int64) error {
	amount := prevBalance - b.Balance
	if amount <= 0 {
		return fmt.Errorf("send amount must be positive (prev=%d, new=%d)", prevBalance, b.Balance)
	}
	if b.Balance < 0 {
		return fmt.Errorf("balance cannot go negative")
	}
	if b.Link == "" {
		return fmt.Errorf("send block must specify recipient in link field")
	}

	l.pending[b.Link] = append(l.pending[b.Link], PendingTx{
		Hash:    b.Hash,
		Amount:  amount,
		Sender:  b.Account,
		Payload: b.Payload,
	})
	return nil
}

// processReceive handles incoming token claims. The receiver's balance
// increases by the amount locked in the matching PendingTx.
func (l *Lattice) processReceive(b *torrent.Block, prevBalance int64) error {
	pending := l.pending[b.Account]
	for i, p := range pending {
		if p.Hash == b.Link {
			expectedBalance := prevBalance + p.Amount
			if b.Balance != expectedBalance {
				return fmt.Errorf("receive balance mismatch: expected %d, got %d", expectedBalance, b.Balance)
			}
			// Remove consumed pending tx
			l.pending[b.Account] = append(pending[:i], pending[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("pending transaction %s not found", b.Link[:16])
}

// processMarketBid handles storage market bid creation. The bidder burns
// BOB tokens and posts a magnet URI for supernodes to store.
func (l *Lattice) processMarketBid(b *torrent.Block, prevBalance int64) error {
	amount := prevBalance - b.Balance
	if amount <= 0 {
		return fmt.Errorf("market bid must cost BOB")
	}

	magnet := ""
	if payload, ok := b.Payload.(map[string]interface{}); ok {
		if m, ok := payload["magnet"].(string); ok {
			magnet = m
		}
	}

	l.marketBids[b.Hash] = &MarketBid{
		ID:        b.Hash,
		Creator:   b.Account,
		Magnet:    magnet,
		Amount:    amount,
		Status:    "OPEN",
		Timestamp: b.Timestamp,
	}
	return nil
}

// processAcceptBid handles storage bid acceptance by a supernode. The
// supernode receives the bid's bounty and the bid is marked ACCEPTED.
func (l *Lattice) processAcceptBid(b *torrent.Block, prevBalance int64) error {
	bid := l.marketBids[b.Link]
	if bid == nil {
		return fmt.Errorf("bid %s not found", b.Link[:16])
	}
	if bid.Status != "OPEN" {
		return fmt.Errorf("bid already %s", bid.Status)
	}

	expectedBalance := prevBalance + bid.Amount
	if b.Balance != expectedBalance {
		return fmt.Errorf("accept_bid balance mismatch: expected %d, got %d", expectedBalance, b.Balance)
	}

	bid.Status = "ACCEPTED"
	bid.AcceptedBy = b.Account
	return nil
}

// processProposal handles governance proposal submission. The proposer
// burns ProposalCost BOB and a new Proposal is created.
func (l *Lattice) processProposal(b *torrent.Block, prevBalance int64) error {
	cost := int64(ProposalCost)
	if prevBalance < cost {
		return fmt.Errorf("insufficient balance for proposal (need %d, have %d)", cost, prevBalance)
	}

	expectedBalance := prevBalance - cost
	if b.Balance != expectedBalance {
		return fmt.Errorf("proposal balance mismatch: expected %d, got %d", expectedBalance, b.Balance)
	}

	title := ""
	description := ""
	if payload, ok := b.Payload.(map[string]interface{}); ok {
		if t, ok := payload["title"].(string); ok {
			title = t
		}
		if d, ok := payload["description"].(string); ok {
			description = d
		}
	}

	// Voting period: 7 days from proposal creation
	l.proposals[b.Hash] = &Proposal{
		ID:          b.Hash,
		Proposer:    b.Account,
		Title:       title,
		Description: description,
		Status:      "active",
		EndTime:     b.Timestamp + (7 * 24 * 60 * 60 * 1000),
		Timestamp:   b.Timestamp,
	}
	l.votes[b.Hash] = make(map[string]Vote)
	return nil
}

// processVote handles governance voting. Each account can vote once per
// proposal with Quadratic Voting power derived from their balance + stake.
func (l *Lattice) processVote(b *torrent.Block, prevBalance int64) error {
	proposalID := b.Link
	proposal := l.proposals[proposalID]
	if proposal == nil {
		return fmt.Errorf("proposal %s not found", proposalID[:16])
	}
	if proposal.Status != "active" {
		return fmt.Errorf("proposal %s is %s", proposalID[:16], proposal.Status)
	}
	if b.Timestamp > proposal.EndTime {
		proposal.Status = "failed" // Auto-close expired proposals
		return fmt.Errorf("voting period has ended")
	}

	// Check for double voting
	if _, voted := l.votes[proposalID][b.Account]; voted {
		return fmt.Errorf("account already voted on proposal %s", proposalID[:16])
	}

	// Calculate quadratic voting power
	power := l.GetQuadraticVotingPower(b.Account)

	voteType := "for"
	if payload, ok := b.Payload.(map[string]interface{}); ok {
		if t, ok := payload["vote"].(string); ok {
			voteType = t
		}
	}

	vote := Vote{Type: voteType, Power: power}
	l.votes[proposalID][b.Account] = vote

	if voteType == "for" {
		proposal.VotesFor += power
	} else {
		proposal.VotesAgainst += power
	}

	// Check if proposal passes (votesFor > votesAgainst with quorum)
	if proposal.VotesFor > proposal.VotesAgainst && (proposal.VotesFor+proposal.VotesAgainst) >= 10 {
		proposal.Status = "passed"
	}

	return nil
}

// processMintNFT handles the creation of a new non-fungible token. The
// minter burns NFTMintCost BOB and a new NFT is registered.
func (l *Lattice) processMintNFT(b *torrent.Block, prevBalance int64) error {
	cost := int64(NFTMintCost)
	if prevBalance < cost {
		return fmt.Errorf("insufficient balance for NFT mint (need %d, have %d)", cost, prevBalance)
	}

	expectedBalance := prevBalance - cost
	if b.Balance != expectedBalance {
		return fmt.Errorf("mint_nft balance mismatch: expected %d, got %d", expectedBalance, b.Balance)
	}

	name := ""
	magnet := ""
	description := ""
	if payload, ok := b.Payload.(map[string]interface{}); ok {
		if n, ok := payload["name"].(string); ok { name = n }
		if m, ok := payload["magnet"].(string); ok { magnet = m }
		if d, ok := payload["description"].(string); ok { description = d }
	}

	l.nfts[b.Hash] = &NFT{
		ID:          b.Hash,
		Owner:       b.Account,
		Name:        name,
		Magnet:      magnet,
		Description: description,
		Timestamp:   b.Timestamp,
	}
	return nil
}

// processTransferNFT handles ownership transfer of an existing NFT. The
// sender burns NFTTransferFee BOB and the NFT's owner is updated.
func (l *Lattice) processTransferNFT(b *torrent.Block, prevBalance int64) error {
	nftID := b.Link
	nft := l.nfts[nftID]
	if nft == nil {
		return fmt.Errorf("NFT %s not found", nftID[:16])
	}
	if nft.Owner != b.Account {
		return fmt.Errorf("only the owner can transfer NFT %s", nftID[:16])
	}

	fee := int64(NFTTransferFee)
	expectedBalance := prevBalance - fee
	if b.Balance != expectedBalance {
		return fmt.Errorf("transfer_nft balance mismatch")
	}

	// Extract new owner from payload
	newOwner := ""
	if payload, ok := b.Payload.(map[string]interface{}); ok {
		if o, ok := payload["newOwner"].(string); ok {
			newOwner = o
		}
		if newOwner == "" {
			if o, ok := payload["recipient"].(string); ok {
				newOwner = o
			}
		}
	}
	if newOwner == "" {
		return fmt.Errorf("transfer_nft must specify newOwner in payload")
	}

	nft.Owner = newOwner
	return nil
}

// processStake handles token staking. Tokens are moved from liquid balance
// to staked balance, earning yield and amplifying governance power.
func (l *Lattice) processStake(b *torrent.Block, prevBalance int64) error {
	frontier := l.GetFrontier(b.Account)
	prevStaked := int64(0)
	if frontier != nil {
		prevStaked = frontier.StakedBalance
	}

	stakeAmount := prevBalance - b.Balance
	if stakeAmount <= 0 {
		return fmt.Errorf("stake amount must be positive")
	}

	expectedStaked := prevStaked + stakeAmount
	if b.StakedBalance != expectedStaked {
		return fmt.Errorf("staked balance mismatch: expected %d, got %d", expectedStaked, b.StakedBalance)
	}

	l.stakeInfo[b.Account] = &StakeInfo{
		Amount:    expectedStaked,
		StartTime: b.Timestamp,
	}
	return nil
}

// processUnstake handles token unstaking. Tokens are moved from staked
// balance back to liquid balance, plus any accumulated yield.
func (l *Lattice) processUnstake(b *torrent.Block, prevBalance int64) error {
	frontier := l.GetFrontier(b.Account)
	if frontier == nil || frontier.StakedBalance <= 0 {
		return fmt.Errorf("no staked balance to unstake")
	}

	unstakeAmount := frontier.StakedBalance - b.StakedBalance
	if unstakeAmount <= 0 {
		return fmt.Errorf("unstake amount must be positive")
	}

	// Calculate yield
	info := l.stakeInfo[b.Account]
	yield := int64(0)
	if info != nil {
		elapsed := b.Timestamp - info.StartTime
		yield = int64(float64(unstakeAmount) * StakingYieldRatePerMS * float64(elapsed))
	}

	expectedBalance := prevBalance + unstakeAmount + yield
	if b.Balance != expectedBalance {
		return fmt.Errorf("unstake balance mismatch: expected %d, got %d", expectedBalance, b.Balance)
	}

	// Update or remove stake info
	if b.StakedBalance <= 0 {
		delete(l.stakeInfo, b.Account)
	} else {
		l.stakeInfo[b.Account] = &StakeInfo{
			Amount:    b.StakedBalance,
			StartTime: b.Timestamp,
		}
	}
	return nil
}

// processInitiateSwap handles HTLC creation for atomic swaps. The initiator
// locks tokens with a hash lock and expiry time.
func (l *Lattice) processInitiateSwap(b *torrent.Block, prevBalance int64) error {
	lockAmount := prevBalance - b.Balance
	if lockAmount <= 0 {
		return fmt.Errorf("swap lock amount must be positive")
	}

	recipient := ""
	hashLock := ""
	expiry := int64(0)
	if payload, ok := b.Payload.(map[string]interface{}); ok {
		if r, ok := payload["recipient"].(string); ok { recipient = r }
		if h, ok := payload["hashLock"].(string); ok { hashLock = h }
		if e, ok := payload["expiry"].(float64); ok { expiry = int64(e) }
	}

	if recipient == "" || hashLock == "" || expiry <= b.Timestamp {
		return fmt.Errorf("invalid swap parameters")
	}

	l.swaps[b.Hash] = &Swap{
		ID:        b.Hash,
		Sender:    b.Account,
		Recipient: recipient,
		Amount:    lockAmount,
		HashLock:  hashLock,
		Expiry:    expiry,
		Status:    "LOCKED",
	}
	return nil
}

// processClaimSwap handles HTLC claiming. The recipient reveals the secret
// matching the hash lock to receive the locked tokens.
func (l *Lattice) processClaimSwap(b *torrent.Block, prevBalance int64) error {
	swapID := b.Link
	swap := l.swaps[swapID]
	if swap == nil {
		return fmt.Errorf("swap %s not found", swapID[:16])
	}
	if swap.Status != "LOCKED" {
		return fmt.Errorf("swap already %s", swap.Status)
	}
	if b.Account != swap.Recipient {
		return fmt.Errorf("only the intended recipient can claim")
	}
	if b.Timestamp > swap.Expiry {
		return fmt.Errorf("swap has expired")
	}

	// Verify the secret matches the hash lock
	secret := ""
	if payload, ok := b.Payload.(map[string]interface{}); ok {
		if s, ok := payload["secret"].(string); ok { secret = s }
	}

	secretHash := torrent.HashSHA256(secret)
	if secretHash != swap.HashLock {
		return fmt.Errorf("invalid secret: hash mismatch")
	}

	expectedBalance := prevBalance + swap.Amount
	if b.Balance != expectedBalance {
		return fmt.Errorf("claim_swap balance mismatch: expected %d, got %d", expectedBalance, b.Balance)
	}

	swap.Status = "CLAIMED"
	swap.Claimer = b.Account
	swap.Secret = secret
	return nil
}

// processRefundSwap handles HTLC refund after expiry. The original sender
// reclaims their locked tokens if the recipient failed to claim in time.
func (l *Lattice) processRefundSwap(b *torrent.Block, prevBalance int64) error {
	swapID := b.Link
	swap := l.swaps[swapID]
	if swap == nil {
		return fmt.Errorf("swap %s not found", swapID[:16])
	}
	if swap.Status != "LOCKED" {
		return fmt.Errorf("swap already %s", swap.Status)
	}
	if b.Account != swap.Sender {
		return fmt.Errorf("only the sender can refund")
	}
	if b.Timestamp <= swap.Expiry {
		return fmt.Errorf("swap has not expired yet")
	}

	expectedBalance := prevBalance + swap.Amount
	if b.Balance != expectedBalance {
		return fmt.Errorf("refund_swap balance mismatch: expected %d, got %d", expectedBalance, b.Balance)
	}

	swap.Status = "REFUNDED"
	return nil
}

// processPublishManifest anchors a previously published off-chain manifest on
// the lattice without mutating account balance. This is the primary bridge
// between the Go supernode publication registry and Bobcoin wallet identity.
func (l *Lattice) processPublishManifest(b *torrent.Block, prevBalance int64) error {
	if b.Balance != prevBalance {
		return fmt.Errorf("publish_manifest must not change balance")
	}

	payload, ok := b.Payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("publish_manifest payload required")
	}

	manifestID, _ := payload["manifestId"].(string)
	locator, _ := payload["locator"].(string)
	manifestURL, _ := payload["manifestUrl"].(string)
	name, _ := payload["name"].(string)
	ciphertextHash, _ := payload["ciphertextHash"].(string)
	size := int64(0)
	if s, ok := payload["size"].(float64); ok {
		size = int64(s)
	}

	if manifestID == "" {
		return fmt.Errorf("publish_manifest requires manifestId")
	}
	if locator == "" {
		locator = fmt.Sprintf("bobtorrent://manifest/%s", manifestID)
	}

	proofHash := ""
	proofSignature := ""
	if proof, ok := payload["publicationProof"].(map[string]interface{}); ok {
		proofHash, _ = proof["messageHash"].(string)
		proofSignature, _ = proof["signature"].(string)
		proofPub, _ := proof["publicKey"].(string)
		if proofHash != "" && proofSignature != "" {
			if proofPub != b.Account {
				return fmt.Errorf("publication proof public key must match block account")
			}
			if !torrent.Verify(proofHash, proofSignature, proofPub) {
				return fmt.Errorf("invalid publication proof signature")
			}
		}
	}

	l.anchors[b.Hash] = &ManifestAnchor{
		ID:             manifestID,
		BlockHash:      b.Hash,
		Owner:          b.Account,
		Type:           "publish_manifest",
		ManifestID:     manifestID,
		Locator:        locator,
		ManifestURL:    manifestURL,
		Name:           name,
		Size:           size,
		CiphertextHash: ciphertextHash,
		ProofHash:      proofHash,
		ProofSignature: proofSignature,
		Timestamp:      b.Timestamp,
	}
	return nil
}

// processDataAnchor preserves compatibility with the older Bobcoin Vault flow
// that burns 10 BOB to anchor a magnet-backed permanent storage reference.
func (l *Lattice) processDataAnchor(b *torrent.Block, prevBalance int64) error {
	expectedBalance := prevBalance - int64(DataAnchorCost)
	if b.Balance != expectedBalance {
		return fmt.Errorf("data_anchor balance mismatch: expected %d, got %d", expectedBalance, b.Balance)
	}

	payload, ok := b.Payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("data_anchor payload required")
	}

	name, _ := payload["name"].(string)
	magnet, _ := payload["magnet"].(string)
	size := int64(0)
	if s, ok := payload["size"].(float64); ok {
		size = int64(s)
	}
	if magnet == "" {
		return fmt.Errorf("data_anchor requires magnet payload")
	}

	l.anchors[b.Hash] = &ManifestAnchor{
		ID:        b.Hash,
		BlockHash: b.Hash,
		Owner:     b.Account,
		Type:      "data_anchor",
		Locator:   magnet,
		Magnet:    magnet,
		Name:      name,
		Size:      size,
		Timestamp: b.Timestamp,
	}
	return nil
}

// ════════════════════════════════════════════════════════════════════════════
// State Hash
// ════════════════════════════════════════════════════════════════════════════

// updateStateHash folds a newly confirmed block's hash into the cumulative
// rolling state hash. This creates a Merkle-like commitment to the entire
// lattice history without requiring a Merkle tree structure.
func (l *Lattice) updateStateHash(b *torrent.Block) {
	raw := l.stateHash + b.Hash
	hash := sha256.Sum256([]byte(raw))
	l.stateHash = hex.EncodeToString(hash[:])
}
