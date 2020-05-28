package natt

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

func (h *Host) CreateStream(ctx context.Context, protocol protocol.ID, peerID peer.ID, forceNew bool) error {
	if !forceNew {
		for _, peerConn := range h.host.Network().Conns() {
			if peerConn.RemotePeer() != peerID {
				continue
			}
			for _, stream := range peerConn.GetStreams() {
				if stream.Protocol() == protocol {
					return fmt.Errorf("%s | protocol:%s | peerID:%s", ErrCreateStreamExist, protocol, peerID)
				}
			}
		}
	}
	h.host.NewStream(ctx, peerID, protocol)
	return nil
}

func (h *Host) SetProtocolStreamHanlder(protocol protocol.ID, handler network.StreamHandler) {
	h.host.RemoveStreamHandler(protocol)
	h.host.SetStreamHandler(protocol, handler)
	// h.host.ConnManager().Notifee().
}
