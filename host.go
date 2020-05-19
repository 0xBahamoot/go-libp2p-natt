package natt

import (
	"context"
	"fmt"
	"io"
	"log"
	"strconv"
	"time"

	mrand "math/rand"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	nat "github.com/libp2p/go-nat"
	"github.com/libp2p/go-tcp-transport"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
)

type Host struct {
	host            host.Host
	natType         network.Reachability
	broadcastAddr   []string
	listenAddrs     []string
	intPort         int
	natDevice       nat.NAT
	cancel          context.CancelFunc
	traversalMethod TraversalMethod
	identityKey     crypto.PrivKey
}

func CreateHost(identityKey crypto.PrivKey, port int, NATdiscoverAddr string, pctx context.Context) (*Host, error) {
	if pctx == nil {
		pctx = context.Background()
	}
	ctx, cancel := context.WithCancel(pctx)

	host := Host{
		natType:       network.ReachabilityUnknown,
		broadcastAddr: []string{},
		intPort:       port,
		cancel:        cancel,
		identityKey:   identityKey,
	}

	natDevice, err := checkNATDevice(ctx)
	if err != nil {
		fmt.Println(err)
	} else {
		host.natDevice = natDevice
	}

	hostAddrs := GetOutboundIP()

	var listenAddrs []string
	for _, addr := range hostAddrs {
		listenAddrs = append(listenAddrs, "/ip4/"+addr+"/tcp/"+strconv.Itoa(port))
	}

	copy(host.listenAddrs, listenAddrs)

	if identityKey == nil {
		var r io.Reader
		r = mrand.New(mrand.NewSource(time.Now().UnixNano()))

		identityKey, _, err = crypto.GenerateKeyPairWithReader(crypto.Ed25519, 0, r)
		if err != nil {
			panic(err)
		}
	}

	h, err := libp2p.New(ctx, libp2p.ListenAddrStrings(listenAddrs...), libp2p.EnableNATService(), libp2p.NATPortMap(), libp2p.Transport(tcp.NewTCPTransport), libp2p.Identity(identityKey))
	if err != nil {
		return nil, err
	}

	host.host = h

	if NATdiscoverAddr != "" {
		serviceInf, err := PeerInfoFromString(NATdiscoverAddr)
		if err != nil {
			return nil, err
		}
		h.Peerstore().AddAddrs(serviceInf.ID, serviceInf.Addrs, time.Hour)
		err = h.Connect(ctx, h.Peerstore().PeerInfo(serviceInf.ID))
		if err != nil {
			return nil, err
		}
		go func() {
			cSub, err := h.EventBus().Subscribe(new(event.EvtLocalReachabilityChanged))
			if err != nil {
				panic(err)
			}
			defer cSub.Close()
			for {
				select {
				case stat := <-cSub.Out():
					if stat == network.ReachabilityUnknown {
						panic("After status update, client did not know its status")
					}
					t := stat.(event.EvtLocalReachabilityChanged)
					host.natType = t.Reachability
					err := host.updateBroadcastAddr()
					if err != nil {
						log.Fatal(err)
					}
				case <-ctx.Done():
					return
				}
			}
		}()

	}
	if err := host.updateBroadcastAddr(); err != nil {
		return nil, err
	}
	return &host, nil
}

func (h *Host) GetNATType() network.Reachability {
	return h.natType
}

func (h *Host) GetBroadcastAddrInfo() []string {
	var result []string
	copy(result, h.broadcastAddr)
	return result
}

func (h *Host) GetHost() host.Host {
	return h.host
}

func (h *Host) Quit() {
	h.cancel()
}

func (h *Host) GetInternalPort() int {
	return h.intPort
}

func (h *Host) updateBroadcastAddr() error {
	switch h.natType {
	case network.ReachabilityUnknown, network.ReachabilityPrivate:
		//behind router that is nested NATs or that not support PCP protocol
		hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", h.host.ID().Pretty()))
		var fullAddr []string
		for _, addr := range h.host.Addrs() {
			fullAddr = append(fullAddr, addr.Encapsulate(hostAddr).String())
		}
		h.broadcastAddr = fullAddr
	case network.ReachabilityPublic:
		if h.natDevice == nil {
			//public IP case
			hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", h.host.ID().Pretty()))
			var fullAddr []string
			for _, addr := range h.host.Addrs() {
				fullAddr = append(fullAddr, addr.Encapsulate(hostAddr).String())
			}
			h.broadcastAddr = fullAddr
		} else {
			//behind public IP router that support PCP protocol
			for _, addr := range h.host.Addrs() {
				if manet.IsPublicAddr(addr) {
					hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", h.host.ID().Pretty()))
					var fullAddr []string
					for _, addr := range h.host.Addrs() {
						fullAddr = append(fullAddr, addr.Encapsulate(hostAddr).String())
					}
					h.broadcastAddr = fullAddr
					return nil
				}
			}

		}
	}
	return nil
}

func (h *Host) GetHostID() peer.ID {
	return h.host.ID()
}

func (h *Host) ConnectPeer(peerAddr string) error {
	return nil
}

func (h *Host) GetListenAddrs() []string {
	var result []string
	copy(result, h.listenAddrs)
	return result
}
