package i2p

import (
	"context"
	"log"
	"net"

	"github.com/go-i2p/i2pkeys"
	"github.com/go-i2p/sam3"
)

type SamSession struct {
	SAMAddr string
	sam     *sam3.SAM
	keys    *i2pkeys.I2PKeys
	stream  *sam3.StreamSession
}

func NewSamSession(addr string) *SamSession {
	if addr == "" {
		addr = "127.0.0.1:7656"
	}
	return &SamSession{
		SAMAddr: addr,
	}
}

func (s *SamSession) Connect() error {
	log.Printf("I2P SAM connecting to %s...", s.SAMAddr)
	sam, err := sam3.NewSAM(s.SAMAddr)
	if err != nil {
		return err
	}
	s.sam = sam

	keys, err := sam.EnsureKeyfile("bobtorrent-keys")
	if err != nil {
		return err
	}
	s.keys = &keys

	stream, err := sam.NewStreamSession("bobtorrent-stream", *s.keys, sam3.Options_Medium)
	if err != nil {
		return err
	}
	s.stream = stream

	log.Printf("I2P SAM Session established. Destination: %s", keys.Addr().Base32())

	return nil
}

// Dial creates a new stream connection to an I2P destination
func (s *SamSession) Dial(ctx context.Context, dest string) (net.Conn, error) {
    return s.stream.Dial("tcp", dest)
}

func (s *SamSession) Close() {
    if s.stream != nil {
        s.stream.Close()
    }
	if s.sam != nil {
		s.sam.Close()
	}
}
