package packet

import (
	"errors"
	"io"
)

// Error value
var ErrInvalidPacketType = errors.New("invalid MQTT Control Packet type")

// Packet represents an MQTT Control Packet.
type Packet interface {
	io.WriterTo
	// Type return the MQTT Control Packet type of the Packet.
	Type() (byte, error)
}

// NewFromBytes creates a Packet from the byte data and returns it.
func NewFromBytes(fixedHeader fixedHeader, remaining []byte) (Packet, error) {
	// Extract the MQTT Control Packet type from the fixed header.
	ptype, err := fixedHeader.ptype()
	if err != nil {
		return nil, err
	}

	var p Packet

	switch ptype {
	default:
		return nil, ErrInvalidPacketType
	}

	// Return the Packet.
	return p, nil
}
