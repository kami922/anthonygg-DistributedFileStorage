package p2p

// Peer is a representation of remote Node in network.
// it contains information about the peer such as its ID,
// address, and other relevant details.
type Peer interface {
	Close() error
}

// Transport is anything that can be used to send and receive messages
//
//	between peers. it is anything that handles communcation
//
// between peers it can be implemented using various protocols such as
// TCP, UDP, or even WebSockets.
type Transport interface {
	ListenAndAccept() error
	Consume() <-chan RPC
}
