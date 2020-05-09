package natt

import (
	"context"
	"errors"

	nat "github.com/libp2p/go-nat"
)

func checkNATDevice(ctx context.Context) (nat.NAT, error) {
	var (
		natInstance nat.NAT
		err         error
	)

	done := make(chan struct{})
	go func() {
		defer close(done)
		// This will abort in 10 seconds anyways.
		natInstance, err = nat.DiscoverGateway()
	}()

	select {
	case <-done:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	if err != nil {
		return nil, err
	}

	_, err = natInstance.GetDeviceAddress()
	if err != nil {
		return nil, errors.New("DiscoverGateway address error:" + err.Error())
	}

	return natInstance, nil
}
