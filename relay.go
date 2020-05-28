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

func (h *Host) findRelayPeers() {
	h.relayPeerCandidate = []peer.ID{}
	for _, peer := range h.peerList {
		if !checkPeerIDExist(h.relayPeerCandidate, peer.ID) {
			canHop, err := circuit.CanHop(h.ctx, h.host, peer.ID)
			if err != nil {
				fmt.Println(err)
			}
			if canHop {
				h.relayPeerCandidate = append(h.relayPeerCandidate, peer.ID)
			}
		}
	}
}

func (h *Host) connectRelayPeer() {
	newRelayPeerCandidate := []peer.ID{}
	for _, peerID := range h.relayPeerCandidate {
		if checkPeerIDExist(h.relayPeerConns, peerID) {
			continue
		}
		if len(h.relayPeerConns) >= maxRelayPeer {
			newRelayPeerCandidate = append(newRelayPeerCandidate, peerID)
			continue
		}
		if checkPeerIDExist(h.currentConnectedPeer, peerID) {
			h.relayPeerConns = append(h.relayPeerConns, peerID)
			continue
		}
		err := h.host.Connect(h.ctx, h.host.Peerstore().PeerInfo(peerID))
		if err != nil {
			fmt.Println(err)
		} else {
			h.relayPeerConns = append(h.relayPeerConns, peerID)
		}
	}
	h.relayPeerCandidate = newRelayPeerCandidate
}

func (h *Host) getCurrentPeerRelay() []peer.ID {
	result := make([]peer.ID, len(h.relayPeerConns))
	copy(result, h.relayPeerConns)
	return result
}
