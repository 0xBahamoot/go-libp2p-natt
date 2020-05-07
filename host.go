package natt

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/libp2p/go-libp2p"
	autonat "github.com/libp2p/go-libp2p-autonat"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
)

type Host struct {
	host          *host.Host
	natType       network.Reachability
	broadcastAddr string
	intPort       int
	natDevice     *NAT
	natMapping    Mapping
	cancel        context.CancelFunc
}

func CreateHost(port int, NATdiscoverAddr string, autoNATService bool) (*Host, error) {
	ctx, cancel := context.WithCancel(context.Background())

	host := Host{
		natType:       network.ReachabilityUnknown,
		broadcastAddr: "",
		intPort:       port,
		cancel:        cancel,
	}
	var hostAddrStr string

	natDevice, err := checkNATDevice()
	if err != nil {
		fmt.Println(err)
	}

	if natDevice != nil {
		host.natDevice = natDevice
		natMapping, err := createMapping(natDevice)
		if err != nil {
			return nil, err
		}
		host.natMapping = natMapping
		hostAddr, err := natDevice.GetInternalAddress()
		if err != nil {
			return nil, err
		}
		hostAddrStr = hostAddr.String()
	}

	if hostAddrStr == "" {
		hostAddr := GetOutboundIP()
		hostAddrStr = hostAddr.String()
	}

	h, err := libp2p.New(ctx, libp2p.ListenAddrStrings("/ip4/"+hostAddrStr+"/tcp/"+strconv.Itoa(port)), libp2p.EnableNATService())
	if err != nil {
		return nil, err
	}
	host.host = &h
	if autoNATService {
		dialback, err := libp2p.New(ctx, libp2p.NoListenAddrs)
		if err != nil {
			return nil, err
		}
		_, err = autonat.New(ctx, h, autonat.EnableService(dialback.Network()))
		if err != nil {
			return nil, err
		}
	}

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
				host.natType = stat.(network.Reachability)
			case <-ctx.Done():
				return
			}
		}()

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

func (h *Host) GetHost() *host.Host {
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
		addrInfo := host.InfoFromHost(*h.host)
		h.broadcastAddr = fmt.Sprintf("/ip4/%s/tcp/%s", addrInfo.Addrs[0].String(), strconv.Itoa(h.intPort))
	case network.ReachabilityPrivate:

	case network.ReachabilityPublic:
	default:
		return ErrorCantUpdateBroadcastAddress
	}
	return nil
}
