package packet

// Defalut values
var (
	DefaultCleanSession      = true
	DefaultKeepAlive    uint = 60
)

// CONNECTOptions represents options for a CONNECT Packet.
type CONNECTOptions struct {
	// CleanSession is the Clean Session of the connect flags.
	CleanSession *bool
	// WillTopic is the Will Topic of the payload.
	WillTopic string
	// WillMessage is the Will Message of the payload.
	WillMessage string
	// WillQoS is the Will QoS of the connect flags.
	WillQoS uint
	// WillRetain is the Will Retain of the connect flags.
	WillRetain bool
	// UserName is the user name used by the server for authentication and authorization.
	UserName string
	// Password is the password used by the server for authentication and authorization.
	Password string
	// KeepAlive is the Keep Alive in the variable header.
	KeepAlive *uint
}

// Init initializes the CONNECTOptions.
func (opts *CONNECTOptions) Init() {
	if opts.CleanSession == nil {
		opts.CleanSession = &DefaultCleanSession
	}

	if opts.KeepAlive == nil {
		opts.KeepAlive = &DefaultKeepAlive
	}
}
