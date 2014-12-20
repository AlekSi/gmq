package packet

// PUBLISHOptions represents options for a PUBLISH Packet.
type PUBLISHOptions struct {
	// DUP is the DUP flag of the Fixed header.
	DUP bool
	// QoS is the QoS of the Fixed header.
	QoS uint
	// Retain is the Retain of the Fixed header.
	Retain bool
	// TopicName is the Topic Name of the Variable header.
	TopicName string
}
