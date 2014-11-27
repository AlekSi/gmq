package client

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/yosssi/gmq/common"
	"github.com/yosssi/gmq/common/packet"
)

// Defalut values
const (
	defaultErrcBufferSize  = 1024
	defaultSendcBufferSize = 1024
)

// Error values
var ErrAlreadyConnected = errors.New("the Client has already connected to the Server")

// Client represents a Client.
type Client struct {
	// Errc is a channel handling errors which are sent by the goroutines
	// which sends or receives MQTT Control Packets.
	Errc chan error
	// mu is a reader/writer mutual exclusion lock for the Client.
	mu sync.RWMutex
	// networkConnection is a Network Connection.
	conn *common.Connection
	// sendc is a channel handling MQTT Control Packets which are sent from
	// the Client to the Server.
	sendc chan packet.Packet
}

// Connect tries to establish a network connection to the Server and
// sends a CONNECT Package to the Server.
func (cli *Client) Connect(address string, opts *packet.CONNECTOptions) error {
	// Lock for the update of the Client's fields.
	cli.mu.Lock()
	defer cli.mu.Unlock()

	// Return an error if the Client has already connected to the Server.
	if cli.conn != nil {
		return ErrAlreadyConnected
	}

	// Connect to the Server and create a Network Connection.
	conn, err := common.NewConnection("tcp", address)
	if err != nil {
		return err
	}
	cli.conn = conn

	// Create a send channel handling MQTT Control Packets and set it to the Client.
	cli.sendc = make(chan packet.Packet, defaultSendcBufferSize)

	// Launch a goroutine which sends MQTT Control Packets to the Server.
	go func() {
		// Create a buffered writer.
		w := bufio.NewWriter(cli.conn)

		// Send MQTT Control Packets.
		for p := range cli.sendc {
			if err := cli.send(w, p); err != nil {
				// Reset the buffered writer.
				w.Reset(cli.conn)
				cli.Errc <- err
				continue
			}
		}
	}()

	// Launch a goroutine which receives MQTT Control Packets from the Server.
	go func() {
		// Create a buffered reader.
		r := bufio.NewReader(cli.conn)

		// Receive MQTT Control Packets from the Server.
		for {
			if err := cli.receive(r); err != nil {
				// Reset the buffered reader.
				r.Reset(cli.conn)
				cli.Errc <- err
				continue
			}
		}
	}()

	// Send the CONNECT Packet to the Server.
	cli.sendc <- packet.NewCONNECT(opts)

	return nil
}

// send sends an MQTT Control Packet to the Server.
func (cli *Client) send(w *bufio.Writer, p packet.Packet) error {
	if _, err := p.WriteTo(w); err != nil {
		return err
	}

	return w.Flush()
}

// receive receives MQTT Control Packets from the Server
func (cli *Client) receive(r *bufio.Reader) error {
	// Get the first byte of the Packet.
	b, err := r.ReadByte()
	if err != nil {
		return err
	}

	// Extract the MQTT Control Packet Type from the first byte.
	packetType := b >> 4

	// Create the Fixed header.
	fixedHeader := []byte{b}

	// Get and decode the Remaining Length.
	var mp uint32 = 1 // multiplier
	var rl uint32     // the Remaining Length
	for {
		b, err = r.ReadByte()
		if err != nil {
			return err
		}

		fixedHeader = append(fixedHeader, b)

		rl += uint32(b&127) * mp

		if b&128 == 0 {
			break
		}

		mp *= 128
	}

	// Create the Remaining (the Variable header and the Payload).
	remaining := make([]byte, rl)

	if rl > 0 {
		if _, err = io.ReadFull(r, remaining); err != nil {
			return err
		}
	}

	switch packetType {
	case packet.TypeCONNACK:
		p, err := packet.NewCONNACKFromBytes(fixedHeader, remaining)
		if err != nil {
			return err
		}

		fmt.Printf("%+v", p)
	}

	return nil
}

// New creates and returns a Client.
func New() *Client {
	return &Client{
		Errc: make(chan error, defaultErrcBufferSize),
	}
}
