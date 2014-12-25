package packet

// Length of the fixed header of the PUBREC Packet
const lenPUBRECFixedHeader = 2

// Length of the variable header of the PUBREC Packet
const lenPUBRECVariableHeader = 2

// pubrec represents a PUBREC Packet.
type pubrec struct {
	base
	// packetID is the Packet Identifier of the variable header.
	packetID uint16
}

// PacketID returns the Packet Identifier of the Packet.
func (p *pubrec) PacketID() uint16 {
	return p.packetID
}

// NewPUBRECFromBytes creates a PUBREC Packet
// from the byte data and returns it.
func NewPUBRECFromBytes(fixedHeader FixedHeader, variableHeader []byte) (Packet, error) {
	// Validate the byte data.
	if err := validatePUBRECBytes(fixedHeader, variableHeader); err != nil {
		return nil, err
	}

	// Decode the Packet Identifier.
	// No error occur because of the precedent validation and
	// the returned error is not be taken care of.
	packetID, _ := decodeUint16(variableHeader)

	// Create a PUBREC Packet.
	p := &pubrec{
		packetID: packetID,
	}

	// Set the fixed header to the Packet.
	p.fixedHeader = fixedHeader

	// Set the variable header to the Packet.
	p.variableHeader = variableHeader

	// Return the Packet.
	return p, nil
}

// validatePUBRECBytes validates the fixed header and the variable header.
func validatePUBRECBytes(fixedHeader FixedHeader, variableHeader []byte) error {
	// Extract the MQTT Control Packet type.
	ptype, err := fixedHeader.ptype()
	if err != nil {
		return err
	}

	// Check the length of the fixed header.
	if len(fixedHeader) != lenPUBRECFixedHeader {
		return ErrInvalidFixedHeaderLen
	}

	// Check the MQTT Control Packet type.
	if ptype != TypePUBREC {
		return ErrInvalidPacketType
	}

	// Check the reserved bits of the fixed header.
	if fixedHeader[0]<<4 != 0x00 {
		return ErrInvalidFixedHeader
	}

	// Check the Remaining Length of the fixed header.
	if fixedHeader[1] != lenPUBRECVariableHeader {
		return ErrInvalidRemainingLength
	}

	// Check the length of the variable header.
	if len(variableHeader) != lenPUBRECVariableHeader {
		return ErrInvalidVariableHeaderLen
	}

	return nil
}
