package natt

import (
	"net"

	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

func peerInfoFromString(peerAddr string) (*peer.AddrInfo, error) {
	pAddr, err := ma.NewMultiaddr(peerAddr)
	if err != nil {
		return nil, err
	}

	pInfo, err := peer.AddrInfoFromP2pAddr(pAddr)
	if err != nil {
		return nil, err
	}
	return pInfo, nil
}

// Get preferred outbound ip of this machine
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}
