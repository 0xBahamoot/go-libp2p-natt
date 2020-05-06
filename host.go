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
	host          host.Host
	NATType       byte
	broadcastAddr string
	intPort       int
	natDevice     *NAT
	natTypeChange chan byte
	natMapping    Mapping
}

func CreateHost(port int, NATdiscoverAddr string, autoNATService bool) *Host {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var host Host
	var hostAddrStr string

	natDevice, err := checkNATDevice()
	if err != nil {
		fmt.Println(err)
	}

	if natDevice != nil {
		host.natDevice = natDevice
		natMapping, err := createMapping(natDevice)
		if err != nil {
			panic(err)
		}
		host.natMapping = natMapping
		hostAddr, err := natDevice.GetInternalAddress()
		if err != nil {
			panic(err)
		}
		hostAddrStr = hostAddr.String()
	}

	if hostAddrStr == "" {
		hostAddr := GetOutboundIP()
		hostAddrStr = hostAddr.String()
	}

	h, err := libp2p.New(ctx, libp2p.ListenAddrStrings("/ip4/"+hostAddrStr+"/tcp/"+strconv.Itoa(port)), libp2p.EnableNATService())
	if err != nil {
		panic(err)
	}

	if autoNATService {
		dialback, err := libp2p.New(ctx, libp2p.NoListenAddrs)
		if err != nil {
			panic(err)
		}
		_, err = autonat.New(ctx, h, autonat.EnableService(dialback.Network()))
		if err != nil {
			panic(err)
		}
	}

	if NATdiscoverAddr != "" {
		serviceInf, err := peerInfoFromString(NATdiscoverAddr)
		if err != nil {
			panic(err)
		}
		h.Peerstore().AddAddrs(serviceInf.ID, serviceInf.Addrs, time.Hour)
		err = h.Connect(ctx, h.Peerstore().PeerInfo(serviceInf.ID))
		if err != nil {
			panic(err)
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
				fmt.Println(stat)
			case <-time.After(30 * time.Second):
				panic("sub timed out.")
			}
		}()

	}

	return &host
}
