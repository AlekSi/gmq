package client

import (
	"errors"
	"io"

	"github.com/yosssi/gmq/mqtt"
	"github.com/yosssi/gmq/mqtt/packet"
)

// Error values
var (
	ErrAlreadyConnected = errors.New("the Client has already connected to the Server")
	ErrNotYetConnected  = errors.New("the Client has not yet connected to the Server")
)

// Client represents a Client.
type Client struct {
	// conn is the Network Connection.
	conn *mqtt.Connection
	// Sess is the Session.
	sess *Session
}

// Connect tries to establish a Network Connection to the Server and
// sends a CONNECT Packet to the Server.
func (cli *Client) Connect(opts *ConnectOptions) error {
	// Return an error if the Client has already connected to the Server.
	if cli.conn != nil {
		return ErrAlreadyConnected
	}

	// Initialize the options.
	if opts == nil {
		opts = &ConnectOptions{}
	}
	opts.Init()

	// Connect to the Server and create a Network Connection.
	conn, err := mqtt.NewConnection(opts.Network, opts.Address)
	if err != nil {
		return err
	}

	// Set the Network Connection to the Client.
	cli.conn = conn

	return nil
}

// Disconnect closes the Network Connection.
func (cli *Client) Disconnect() error {
	// Return an error if the Client has not yet connected to the Server.
	if cli.conn == nil {
		return ErrNotYetConnected
	}

	// Close the Network Connection.
	if err := cli.conn.Close(); err != nil {
		return err
	}

	// Clear the Network Connection of the Client.
	cli.conn = nil

	// Clear the Session if the CleanSession is true.
	if cli.sess != nil && cli.sess.CleanSession {
		cli.sess = nil
	}

	return nil
}

// SendCONNECT sends a CONNECT Packet to the Server.
func (cli *Client) SendCONNECT(opts *packet.CONNECTOptions) error {
	// Return an error if the Client has not yet connected to the Server.
	if cli.conn == nil {
		return ErrNotYetConnected
	}

	// Initialize the options.
	if opts == nil {
		opts = &packet.CONNECTOptions{}
	}
	opts.Init()

	// Create a Session or reuse the current Session.
	if *opts.CleanSession || cli.sess == nil {
		// Craete a Session and set it to the Client.
		cli.sess = NewSession(&SessionOptions{
			CleanSession: opts.CleanSession,
			ClientID:     opts.ClientID,
		})
	} else {
		// Reuse the Session and set its Client Identifier to the Packet options.
		opts.ClientID = cli.sess.ClientID
	}

	// Create a CONNECT Packet.
	p, err := packet.NewCONNECT(opts)
	if err != nil {
		return err
	}

	// Send the CONNECT Packet to the Server.
	return cli.send(p)
}

// SendDISCONNECT sends a DISCONNECT Packet to the Server.
func (cli *Client) SendDISCONNECT() error {
	// Return an error if the Client has not yet connected to the Server.
	if cli.conn == nil {
		return ErrNotYetConnected
	}

	// Send a DISCONNECT Packet to the Server.
	return cli.send(packet.NewDISCONNECT())
}

// Receive receives an MQTT Control Packet from the Server.
func (cli *Client) Receive() (byte, packet.Packet, error) {
	// Get the first byte of the Packet.
	b, err := cli.conn.R.ReadByte()
	if err != nil {
		return 0x00, nil, err
	}

	// Extract the MQTT Control Packet Type from the first byte.
	ptype := b >> 4

	// Create the Fixed header.
	fixedHeader := []byte{b}

	// Get and decode the Remaining Length.
	var mp uint32 = 1 // multiplier
	var rl uint32     // the Remaining Length
	for {
		b, err = cli.conn.R.ReadByte()
		if err != nil {
			return 0x00, nil, err
		}

		fixedHeader = append(fixedHeader, b)

		rl += uint32(b&0x7F) * mp

		if b&0x80 == 0 {
			break
		}

		mp *= 128
	}

	// Create the Remaining (the Variable header and the Payload).
	remaining := make([]byte, rl)

	if rl > 0 {
		if _, err = io.ReadFull(cli.conn.R, remaining); err != nil {
			return 0x00, nil, err
		}
	}

	var p packet.Packet

	switch ptype {
	case packet.TypeCONNACK:
		// Create the CONNACK Packet from the byte data to validate the data.
		if p, err = packet.NewCONNACKFromBytes(fixedHeader, remaining); err != nil {
			return 0x00, nil, err
		}
	}

	return ptype, p, nil
}

// send sends an MQTT Control Packet to the Server.
func (cli *Client) send(p packet.Packet) error {
	if _, err := p.WriteTo(cli.conn.W); err != nil {
		return err
	}

	return cli.conn.W.Flush()
}

// New creates and returns a Client.
func New(_ *Options) *Client {
	return &Client{}
}
