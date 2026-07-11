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

// I2PDatagramTransport provides native I2P/SAM datagram support for anonymous signaling.
type I2PDatagramTransport struct {
	sam     *sam3.SAM
	session *sam3.DatagramSession
	keys    i2pkeys.I2PKeys
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewI2PDatagramTransport(samAddr string) (*I2PDatagramTransport, error) {
	ctx, cancel := context.WithCancel(context.Background())
	s, err := sam3.NewSAM(samAddr)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to connect to SAM: %w", err)
	}
	keys, err := s.NewKeys()
	if err != nil {
		s.Close(); cancel()
		return nil, fmt.Errorf("failed to generate I2P keys: %w", err)
	}
	session, err := s.NewDatagramSession("BOBTORRENT-SIG", keys, sam3.Options_Default, 0)
	if err != nil {
		s.Close(); cancel()
		return nil, fmt.Errorf("failed to create I2P datagram session: %w", err)
	}
	log.Printf("I2P Datagram Transport active: %s", keys.Addr().Base32())
	return &I2PDatagramTransport{sam: s, session: session, keys: keys, ctx: ctx, cancel: cancel}, nil
}

func (t *I2PDatagramTransport) SendTo(data []byte, dest i2pkeys.I2PAddr) error {
	_, err := t.session.WriteTo(data, dest)
	return err
}

func (t *I2PDatagramTransport) ReceiveLoop(handler func([]byte, net.Addr)) {
	go func() {
		buf := make([]byte, 32*1024)
		for {
			_ = t.session.SetReadDeadline(time.Now().Add(1 * time.Second))
			n, addr, err := t.session.ReadFrom(buf)
			if err != nil {
				select {
				case <-t.ctx.Done(): return
				default: continue
				}
			}
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])
				handler(data, addr)
			}
		}
	}()
}

func (t *I2PDatagramTransport) Close() error {
	t.cancel()
	t.session.Close()
	return t.sam.Close()
}

func (t *I2PDatagramTransport) LocalAddr() i2pkeys.I2PAddr { return t.keys.Addr() }
