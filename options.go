package natt

import (
	"github.com/libp2p/go-libp2p-core/crypto"
)

type Option struct {
	IdentityKey     crypto.PrivKey
	Port            int
	NATdiscoverAddr string
	EnableRelay     bool
	UseRelayPeer    bool
}
