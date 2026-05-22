package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Messenger manages the libp2p gossip mesh for decentralized chat and signaling.
//
// Why this exists:
//   1. Bobtorrent needs a decentralized messaging layer beyond trackers.
//   2. libp2p GossipSub provides a robust, scalable gossip mesh.
//   3. This allows the 'Mega-Messenger' vision to proceed without a central Matrix server.
type Messenger struct {
	host   host.Host
	pubsub *pubsub.PubSub
	ctx    context.Context
	cancel context.CancelFunc

	topicsMu sync.RWMutex
	topics   map[string]*pubsub.Topic
	subs     map[string]*pubsub.Subscription
	handlers map[string]map[string]func([]byte, peer.ID) // topic -> handlerID -> handler
}

// NewMessenger initializes a libp2p host and a GossipSub engine.
func NewMessenger(listenAddr string) (*Messenger, error) {
	ctx, cancel := context.WithCancel(context.Background())

	h, err := libp2p.New(
		libp2p.ListenAddrStrings(listenAddr),
		libp2p.NATPortMap(), // Attempt to open ports via UPnP/NAT-PMP
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		h.Close()
		cancel()
		return nil, fmt.Errorf("failed to create gossipsub: %w", err)
	}

	log.Printf("Messenger libp2p host started: ID=%s Addrs=%v", h.ID(), h.Addrs())

	return &Messenger{
		host:     h,
		pubsub:   ps,
		ctx:      ctx,
		cancel:   cancel,
		topics:   make(map[string]*pubsub.Topic),
		subs:     make(map[string]*pubsub.Subscription),
		handlers: make(map[string]map[string]func([]byte, peer.ID)),
	}, nil
}

// JoinTopic joins a GossipSub topic and registers a handler for incoming messages.
// Multiple handlers can be registered for the same topic using unique handlerIDs.
func (m *Messenger) JoinTopic(topicName string, handlerID string, handler func([]byte, peer.ID)) error {
	m.topicsMu.Lock()
	defer m.topicsMu.Unlock()

	if m.handlers[topicName] == nil {
		m.handlers[topicName] = make(map[string]func([]byte, peer.ID))
	}
	m.handlers[topicName][handlerID] = handler

	if _, exists := m.topics[topicName]; exists {
		return nil
	}

	t, err := m.pubsub.Join(topicName)
	if err != nil {
		return fmt.Errorf("failed to join topic %s: %w", topicName, err)
	}

	sub, err := t.Subscribe()
	if err != nil {
		t.Close()
		return fmt.Errorf("failed to subscribe to topic %s: %w", topicName, err)
	}

	m.topics[topicName] = t
	m.subs[topicName] = sub

	go m.readLoop(sub)

	return nil
}

// LeaveTopic unregisters a handler from a topic using its handlerID.
func (m *Messenger) LeaveTopic(topicName string, handlerID string) {
	m.topicsMu.Lock()
	defer m.topicsMu.Unlock()

	if m.handlers[topicName] != nil {
		delete(m.handlers[topicName], handlerID)
		if len(m.handlers[topicName]) == 0 {
			delete(m.handlers, topicName)
			// Optional: leave the topic if no handlers remain
			if sub, ok := m.subs[topicName]; ok {
				sub.Cancel()
				delete(m.subs, topicName)
			}
			if topic, ok := m.topics[topicName]; ok {
				topic.Close()
				delete(m.topics, topicName)
			}
		}
	}
}

// UnregisterAllHandlers removes all handlers for a specific topic for a given client identification (stub).
func (m *Messenger) UnregisterAllHandlers(topicName string) {
	m.topicsMu.Lock()
	defer m.topicsMu.Unlock()
	delete(m.handlers, topicName)
	// We keep the topic joined to the mesh for other potential local handlers,
	// unless we really want to leave the mesh for this topic.
}

// Publish broadcasts a message to a GossipSub topic.
func (m *Messenger) Publish(topicName string, data []byte) error {
	m.topicsMu.RLock()
	t, exists := m.topics[topicName]
	m.topicsMu.RUnlock()

	if !exists {
		return fmt.Errorf("not joined to topic %s", topicName)
	}

	return t.Publish(m.ctx, data)
}

func (m *Messenger) readLoop(sub *pubsub.Subscription) {
	topicName := sub.Topic()
	for {
		msg, err := sub.Next(m.ctx)
		if err != nil {
			if m.ctx.Err() != nil {
				return
			}
			log.Printf("Subscription error for topic %s: %v", topicName, err)
			return
		}

		// Don't echo back our own messages
		if msg.ReceivedFrom == m.host.ID() {
			continue
		}

		m.topicsMu.RLock()
		handlers := m.handlers[topicName]
		m.topicsMu.RUnlock()

		for _, h := range handlers {
			go h(msg.Data, msg.ReceivedFrom)
		}
	}
}

// Close shuts down the libp2p host and all subscriptions.
func (m *Messenger) Close() error {
	m.cancel()
	m.topicsMu.Lock()
	for _, sub := range m.subs {
		sub.Cancel()
	}
	for _, t := range m.topics {
		t.Close()
	}
	m.topicsMu.Unlock()
	return m.host.Close()
}

// MatrixEvent represents a minimal Matrix-compatible event envelope for libp2p gossip.
type MatrixEvent struct {
	Type     string                 `json:"type"`
	Sender   string                 `json:"sender"`
	RoomID   string                 `json:"room_id"`
	Content  map[string]interface{} `json:"content"`
	EventID  string                 `json:"event_id,omitempty"`
	OriginTS int64                  `json:"origin_server_ts,omitempty"`
}

// PublishMatrixEvent wraps a Matrix-style event into a GossipSub payload.
func (m *Messenger) PublishMatrixEvent(topicName string, event MatrixEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return m.Publish(topicName, data)
}

// Host returns the underlying libp2p host.
func (m *Messenger) Host() host.Host {
	return m.host
}
