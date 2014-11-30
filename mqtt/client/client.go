package client

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/yosssi/gmq/mqtt"
	"github.com/yosssi/gmq/mqtt/packet"
)

// Defalut values
const (
	defaultErrcBufferSize  = 1024
	defaultSendcBufferSize = 1024
)

// Error string
const strErrHandlingErr = "error %q occurred while handing the error %q"

// Error values
var (
	ErrAlreadyConnected = errors.New("the Client has already connected to the Server")
	ErrNotYetConnected  = errors.New("the Client has not yet connected to the Server")
	ErrNotCONNACK       = errors.New("the Packet which was not the CONNACK Packet has been received")
	ErrCONNACKTimeout   = errors.New("Timeout has occurred while waiting for receiving the CONNACK Packet from the Server")
)

// Client represents a Client.
type Client struct {
	// Errc is a channel handling errors which are sent by the goroutines
	// which sends or receives MQTT Control Packets.
	Errc chan error
	// mu is a reader/writer mutual exclusion lock for the Client.
	mu sync.RWMutex
	// conn is a Network Connection.
	conn *mqtt.Connection
	// sendc is a channel handling MQTT Control Packets which are sent from
	// the Client to the Server.
	sendc chan packet.Packet
	// sess is a Session.
	sess *mqtt.Session
}

// Connect tries to establish a network connection to the Server and
// sends a CONNECT Package to the Server.
func (cli *Client) Connect(opts *ConnectOptions, connectOpts *packet.CONNECTOptions) error {
	// Lock for the update of the Client's fields.
	cli.mu.Lock()
	defer cli.mu.Unlock()

	// Return an error if the Client has already connected to the Server.
	if cli.conn != nil {
		return ErrAlreadyConnected
	}

	// Initialize the options.
	if opts == nil {
		opts = &ConnectOptions{}
	}
	opts.Init()

	// Initialize the options for the CONNECT Packet.
	if connectOpts == nil {
		connectOpts = &packet.CONNECTOptions{}
	}
	connectOpts.Init()

	// Connect to the Server and create a Network Connection.
	conn, err := mqtt.NewConnection(opts.Network, opts.Address)
	if err != nil {
		return err
	}
	cli.conn = conn

	// Create a Session or reuse the current Session.
	if *connectOpts.CleanSession || cli.sess == nil || cli.sess.CleanSession {
		// Craete a Session and set it to the Client.
		cli.sess = mqtt.NewSession(&mqtt.SessionOptions{
			CleanSession: connectOpts.CleanSession,
			ClientID:     connectOpts.ClientID,
		})
	} else {
		// Reuse the Session and set its Client Identifier to the options.
		connectOpts.ClientID = cli.sess.ClientID
	}

	// Send the CONNECT Packet to the Server.
	if err := cli.send(packet.NewCONNECT(connectOpts)); err != nil {
		// Disconnect the Network Connection.
		if anotherErr := cli.disconnect(); anotherErr != nil {
			return fmt.Errorf(strErrHandlingErr, anotherErr, err)
		}

		return err
	}

	// Wait for receiving the CONNACK Packet.
	connacked := make(chan struct{})
	errc := make(chan error)

	go func() {
		p, err := receive(cli.conn.R)
		if err != nil {
			errc <- err
		}

		if _, ok := p.(*packet.CONNACK); !ok {
			errc <- ErrNotCONNACK
		}

		connacked <- struct{}{}
	}()

	select {
	case <-connacked:
	case err := <-errc:
		// Disconnect the Network Connection.
		if anotherErr := cli.disconnect(); anotherErr != nil {
			return fmt.Errorf(strErrHandlingErr, anotherErr, err)
		}
		return err
	case <-time.After(opts.CONNACKTimeout):
		// Disconnect the Network Connection.
		if anotherErr := cli.disconnect(); anotherErr != nil {
			return fmt.Errorf(strErrHandlingErr, anotherErr, ErrCONNACKTimeout)
		}
		return ErrCONNACKTimeout
	}

	// Create a send channel handling MQTT Control Packets and set it to the Client.
	cli.sendc = make(chan packet.Packet, defaultSendcBufferSize)

	// Launch a goroutine which sends MQTT Control Packets to the Server.
	go func() {
		// Send MQTT Control Packets.
		for p := range cli.sendc {
			// Lock for the update of the Client's fields.
			cli.mu.Lock()

			if err := cli.send(p); err != nil {
				// Disconnect the Network Connection.
				if anotherErr := cli.disconnect(); anotherErr != nil {
					cli.Errc <- fmt.Errorf(strErrHandlingErr, anotherErr, ErrCONNACKTimeout)
				} else {
					cli.Errc <- err
				}

				cli.mu.Unlock()

				break
			}

			cli.mu.Unlock()
		}
	}()

	// Launch a goroutine which receives MQTT Control Packets from the Server.
	go func() {
		// Receive MQTT Control Packets from the Server.
		for {
			var r *bufio.Reader

			cli.mu.RLock()

			if cli.conn == nil {
				cli.mu.RUnlock()
				break
			}

			r = cli.conn.R

			cli.mu.RUnlock()

			if _, err := receive(r); err != nil {
				// Lock for the update of the Client's fields.
				cli.mu.Lock()

				if cli.conn == nil {
					cli.mu.Unlock()
					break
				}

				// Disconnect the Network Connection.
				if anotherErr := cli.Disconnect(); anotherErr != nil {
					cli.Errc <- fmt.Errorf(strErrHandlingErr, anotherErr, ErrCONNACKTimeout)
					cli.mu.Unlock()
					break
				}

				cli.Errc <- err
				cli.mu.Unlock()
				break
			}
		}
	}()

	return nil
}

// Disconnect sends the DISCONNECT Packet to the Server and
// closes the Network Connection.
func (cli *Client) Disconnect() error {
	// Lock for the update of the Client's fields.
	cli.mu.Lock()
	defer cli.mu.Unlock()

	// Disconnect the Network Connection.
	err := cli.disconnect()

	return err
}

// send sends an MQTT Control Packet to the Server.
func (cli *Client) send(p packet.Packet) error {
	if _, err := p.WriteTo(cli.conn.W); err != nil {
		return err
	}

	return cli.conn.W.Flush()
}

// receive receives MQTT Control Packets from the Server
func receive(r *bufio.Reader) (packet.Packet, error) {
	// Get the first byte of the Packet.
	b, err := r.ReadByte()
	if err != nil {
		return nil, err
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
			return nil, err
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
			return nil, err
		}
	}

	var p packet.Packet

	switch packetType {
	case packet.TypeCONNACK:
		// Create the CONNACK Packet from the byte data to validate the data.
		if p, err = packet.NewCONNACKFromBytes(fixedHeader, remaining); err != nil {
			return nil, err
		}
	}

	return p, nil
}

// disconnect sends the DISCONNECT Packet to the Server and
// closes the Network Connection.
func (cli *Client) disconnect() error {
	// Return an error if the Client has not yet connected to the Server.
	if cli.conn == nil {
		return ErrNotYetConnected
	}

	// Send the DISCONNECT Packet to the Server.
	if err := cli.send(packet.NewDISCONNECT()); err != nil {
		return err
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

// New creates and returns a Client.
func New() *Client {
	return &Client{
		Errc: make(chan error, defaultErrcBufferSize),
	}
}
