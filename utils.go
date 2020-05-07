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
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		panic(ErrShouldHaveIPAddress.Error())
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP
			}
		}
	}
	panic(ErrShouldHaveIPAddress.Error())
}
