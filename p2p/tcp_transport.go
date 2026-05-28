package p2p

import (
	"fmt"
	"net"
)

type TCPPeer struct {
	conn net.Conn

	outbound bool
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		conn:     conn,
		outbound: outbound,
	}

}

// it implements Peer interface which will close the connection to peer
// when called.
func (p *TCPPeer) Close() error {
	return p.conn.Close()
}

type TCPTransportOpts struct {
	ListenAddr    string
	HandshakeFunc HandshakeFunc
	Decoder       Decoder
	OnPeer        func(Peer) error
}

type TCPTransport struct {
	TCPTransportOpts
	listener net.Listener
	rpcch    chan RPC
}

func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: opts,
		rpcch:            make(chan RPC, 1024),
	}
}

// it implements Transport Inteface which will return read only channel
// for reading incomming messages from peers in network.
func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcch
}

func (t *TCPTransport) ListenAndAccept() error {

	var err error
	t.listener, err = net.Listen("tcp", t.ListenAddr)
	if err != nil {
		return err
	}
	go t.startAcceptLoop()
	return nil

}

func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			fmt.Printf("TCP Accept Error:%s\n", err)
		}

		go t.handleConn(conn)
	}

}

type Temp struct{}

func (t *TCPTransport) handleConn(conn net.Conn) {
	var err error
	defer func() {
		fmt.Printf("dropping peer connection %s", err)
		conn.Close()
	}()

	peer := NewTCPPeer(conn, true)

	if err := t.HandshakeFunc(peer); err != nil {
		return
	}

	if t.OnPeer != nil {
		if err := t.OnPeer(peer); err != nil {
			return
		}
	}

	// Read loop
	for {
		rpc := RPC{}
		err = t.Decoder.Decode(conn, &rpc)
		if err != nil {
			return
		}

		// rpc.From = conn.RemoteAddr().String()

		// if rpc.Stream {
		// 	peer.wg.Add(1)
		// 	fmt.Printf("[%s] incoming stream, waiting...\n", conn.RemoteAddr())
		// 	peer.wg.Wait()
		// 	fmt.Printf("[%s] stream closed, resuming read loop\n", conn.RemoteAddr())
		// 	continue
		// }
		rpc.From = conn.RemoteAddr()
		t.rpcch <- rpc
	}
}
