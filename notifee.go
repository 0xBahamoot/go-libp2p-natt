package natt

import (
	"fmt"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

type notifee struct {
	OnPeerConnected    func(peer.ID)
	OnPeerDisconnected func(peer.ID)
	OnPeerStreamOpened func(peer.ID, network.Stream)
	OnPeerStreamClosed func(peer.ID, network.Stream)
}

// called when network starts listening on an addr
func (n *notifee) Listen(network.Network, ma.Multiaddr) {
	fmt.Println("notifee Listen called")
}

// called when network stops listening on an addr
func (n *notifee) ListenClose(network.Network, ma.Multiaddr) {
	fmt.Println("notifee ListenClose called")
}

// called when a network.connection opened
func (n *notifee) Connected(nw network.Network, conn network.Conn) {
	if n.OnPeerConnected != nil {
		n.OnPeerConnected(conn.RemotePeer())
	}
}

// called when a connection closed
func (n *notifee) Disconnected(nw network.Network, conn network.Conn) {
	if n.OnPeerDisconnected != nil {
		n.OnPeerDisconnected(conn.RemotePeer())
	}
}

// called when a stream opened
func (n *notifee) OpenedStream(nw network.Network, strm network.Stream) {
	if n.OnPeerStreamOpened != nil {
		n.OnPeerStreamOpened(strm.Conn().RemotePeer(), strm)
	}
}

// called when a stream closed
func (n *notifee) ClosedStream(nw network.Network, strm network.Stream) {
	if n.OnPeerStreamClosed != nil {
		n.OnPeerStreamClosed(strm.Conn().RemotePeer(), strm)
	}
}
