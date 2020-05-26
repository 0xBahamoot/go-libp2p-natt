package natt

import (
	"fmt"

	circuit "github.com/libp2p/go-libp2p-circuit"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

func (h *Host) createRelayAddresses() []string {
	var result []string
	for _, peerID := range h.relayPeerConns {
		relayaddr, err := ma.NewMultiaddr("/p2p/" + peerID.Pretty() + "/p2p-circuit/p2p/" + h.host.ID().Pretty())
		if err != nil {
			panic(err)
		}
		result = append(result, relayaddr.String())
	}
	return result
}

func (h *Host) lookforRelayPeers() {
	h.relayPeerCandidate = []peer.ID{}
	for _, peer := range h.peerList {
		canHop, err := circuit.CanHop(h.ctx, h.host, peer.ID)
		if err != nil {
			fmt.Println(err)
		}
		if canHop {
			h.relayPeerCandidate = append(h.relayPeerCandidate, peer.ID)
		}
	}
}

func (h *Host) connectRelayPeer() {
	for _, peerID := range h.relayPeerCandidate {
		err := h.host.Connect(h.ctx, h.host.Peerstore().PeerInfo(peerID))
		if err != nil {
			fmt.Println(err)
		} else {
			h.relayPeerConns = append(h.relayPeerConns, peerID)
		}
		// if h.relayPeerConnCount > maxRelayPeer {
		// 	return
		// }
	}
}

func (h *Host) getCurrentPeerRelay() []peer.ID {
	result := make([]peer.ID, len(h.relayPeerConns))
	copy(result, h.relayPeerConns)
	return result
}
