package natt

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

type Host struct {
	host          host.Host
	natType       network.Reachability
	broadcastAddr string
	intPort       int
	natDevice     *NAT
	natMapping    Mapping
	cancel        context.CancelFunc
}

func CreateHost(port int, NATdiscoverAddr string) (*Host, error) {
	ctx, cancel := context.WithCancel(context.Background())

	host := Host{
		natType:       network.ReachabilityUnknown,
		broadcastAddr: "",
		intPort:       port,
		cancel:        cancel,
	}

	natDevice, err := checkNATDevice()
	if err != nil {
		fmt.Println(err)
	} else {
		host.natDevice = natDevice
	}

	hostAddr := GetOutboundIP()
	hostAddrStr := hostAddr.String()

	h, err := libp2p.New(ctx, libp2p.ListenAddrStrings("/ip4/"+hostAddrStr+"/tcp/"+strconv.Itoa(port)), libp2p.EnableNATService(), libp2p.NATPortMap())
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

func (h *Host) GetNATDevice() *NAT {
	return h.natDevice
}

func (h *Host) GetHost() host.Host {
	return h.host
}

func (h *Host) GetMapping() Mapping {
	return h.natMapping
}

func (h *Host) Quit() {
	h.cancel()
}

func (h *Host) GetInternalPort() int {
	return h.intPort
}

func (h *Host) updateBroadcastAddr() error {
	switch h.natType {
	case network.ReachabilityUnknown:
		return ErrCantUpdateBroadcastAddress
	case network.ReachabilityPrivate:
		//behind router that is nested NATs or that not support PCP protocol
		addrInfo := host.InfoFromHost(h.host)
		hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", addrInfo.ID.Pretty()))
		fullAddr := addrInfo.Addrs[0].Encapsulate(hostAddr)
		h.broadcastAddr = fullAddr.String()
	case network.ReachabilityPublic:
		if h.natDevice == nil {
			//public IP case
			addrInfo := host.InfoFromHost(h.host)
			hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", addrInfo.ID.Pretty()))
			fullAddr := addrInfo.Addrs[0].Encapsulate(hostAddr)
			h.broadcastAddr = fullAddr.String()
		} else {
			//behind public IP router that support PCP protocol
			extAddr, err := h.natMapping.ExternalAddr()
			if err != nil {
				return ErrCantGetExternalAddress
			}
			addrInfo := host.InfoFromHost(h.host)
			addr := fmt.Sprintf("/ip4/%s/tcp/%s/p2p/%s", strings.Split(extAddr.String(), ":")[0], strconv.Itoa(h.natMapping.ExternalPort()), addrInfo.ID.Pretty())
			fullAddr, _ := ma.NewMultiaddr(addr)
			h.broadcastAddr = fullAddr.String()
		}
	}
	return nil
}

func (h *Host) GetHostID() peer.ID {
	return h.host.ID()
}
