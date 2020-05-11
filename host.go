package natt

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

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
	broadcastAddr   string
	intPort         int
	natDevice       nat.NAT
	cancel          context.CancelFunc
	traversalMethod TraversalMethod
	identityKey     crypto.PrivKey
}

func CreateHost(identityKey crypto.PrivKey, port int, NATdiscoverAddr string) (*Host, error) {
	ctx, cancel := context.WithCancel(context.Background())

	host := Host{
		natType:       network.ReachabilityUnknown,
		broadcastAddr: "",
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

	hostAddr := GetOutboundIP()
	hostAddrStr := hostAddr.String()
	if identityKey == nil {

	}

	h, err := libp2p.New(ctx, libp2p.ListenAddrStrings("/ip4/"+hostAddrStr+"/tcp/"+strconv.Itoa(port)), libp2p.EnableNATService(), libp2p.NATPortMap(), libp2p.Transport(tcp.NewTCPTransport), libp2p.Identity(identityKey))
	if err != nil {
		return nil, err
	}

	host.host = h

	if NATdiscoverAddr != "" {
		serviceInf, err := peerInfoFromString(NATdiscoverAddr)
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

func (h *Host) GetBroadcastAddrInfo() string {
	return h.broadcastAddr
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
	// case :
	// 	return ErrCantUpdateBroadcastAddress
	case network.ReachabilityUnknown, network.ReachabilityPrivate:
		//behind router that is nested NATs or that not support PCP protocol
		hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", h.host.ID().Pretty()))
		fullAddr := h.host.Addrs()[0].Encapsulate(hostAddr)
		h.broadcastAddr = fullAddr.String()
	case network.ReachabilityPublic:
		if h.natDevice == nil {
			//public IP case
			hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", h.host.ID().Pretty()))
			fullAddr := h.host.Addrs()[0].Encapsulate(hostAddr)
			h.broadcastAddr = fullAddr.String()
		} else {
			//behind public IP router that support PCP protocol
			for _, addr := range h.host.Addrs() {
				if manet.IsPublicAddr(addr) {
					hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", h.host.ID().Pretty()))
					fullAddr := addr.Encapsulate(hostAddr)
					h.broadcastAddr = fullAddr.String()
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
