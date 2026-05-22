package transport

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/go-i2p/i2pkeys"
	"github.com/go-i2p/sam3"
)

// I2PDatagramTransport provides native I2P/SAM datagram support for anonymous, low-latency signaling.
//
// Why this exists:
//   1. Phase 8 requires integrating native I2P/SAM Datagrams directly into the networking layer.
//   2. Datagrams (UDP-like) are preferred over Stream (TCP-like) for real-time signaling/gossip.
//   3. This complements the 'Shadow Swarms' vision of extreme censorship resistance.
type I2PDatagramTransport struct {
	sam     *sam3.SAM
	session *sam3.DatagramSession
	keys    i2pkeys.I2PKeys
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewI2PDatagramTransport initializes an I2P/SAM datagram session.
// It expects a running I2P router with SAM enabled (usually on localhost:7656).
func NewI2PDatagramTransport(samAddr string) (*I2PDatagramTransport, error) {
	ctx, cancel := context.WithCancel(context.Background())

	s, err := sam3.NewSAM(samAddr)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to connect to SAM: %w", err)
	}

	keys, err := s.NewKeys()
	if err != nil {
		s.Close()
		cancel()
		return nil, fmt.Errorf("failed to generate I2P keys: %w", err)
	}

	// Create a new datagram session.
	// "BOBTORRENT-DATAGRAM" is the destination name.
	session, err := s.NewDatagramSession("BOBTORRENT-DATAGRAM", keys, sam3.Options_Default, 0)
	if err != nil {
		s.Close()
		cancel()
		return nil, fmt.Errorf("failed to create I2P datagram session: %w", err)
	}

	log.Printf("I2P Datagram Transport started: Dest=%s", keys.Addr().Base32())

	return &I2PDatagramTransport{
		sam:     s,
		session: session,
		keys:    keys,
		ctx:     ctx,
		cancel:  cancel,
	}, nil
}

// SendTo sends a datagram to a remote I2P destination.
func (t *I2PDatagramTransport) SendTo(data []byte, dest i2pkeys.I2PAddr) error {
	_, err := t.session.WriteTo(data, dest)
	return err
}

// ReceiveLoop starts a background goroutine to handle incoming datagrams.
func (t *I2PDatagramTransport) ReceiveLoop(handler func([]byte, net.Addr)) {
	go func() {
		buf := make([]byte, 32*1024) // I2P datagrams can be up to 32KB
		for {
			// Set a read deadline so we can check for context cancellation
			_ = t.session.SetReadDeadline(time.Now().Add(1 * time.Second))

			n, addr, err := t.session.ReadFrom(buf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					// Check if context is cancelled on timeout
					select {
					case <-t.ctx.Done():
						return
					default:
						continue
					}
				}
				log.Printf("I2P Datagram receive error: %v", err)
				continue
			}

			select {
			case <-t.ctx.Done():
				return
			default:
				if n > 0 {
					handler(buf[:n], addr)
				}
			}
		}
	}()
}

// Close shuts down the I2P/SAM session.
func (t *I2PDatagramTransport) Close() error {
	t.cancel()
	t.session.Close()
	return t.sam.Close()
}

// LocalAddr returns the local I2P destination address.
func (t *I2PDatagramTransport) LocalAddr() i2pkeys.I2PAddr {
	return t.keys.Addr()
}
