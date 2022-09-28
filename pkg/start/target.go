package start

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/host"
)

type PinningServiceTargetConstructor = func(host host.Host, auth string) (Target, error)

var PinningServiceTargetConstructors = map[string]PinningServiceTargetConstructor{
	InfuraTargetName: NewInfura,
	PinataTargetName: NewPinata,
}

type Target interface {
	Operation(ctx context.Context, c cid.Cid) error
	Backoff(ctx context.Context) backoff.BackOff
	CleanUp(ctx context.Context, c cid.Cid) error
	Timeout() time.Duration
	Rate() time.Duration
	Name() string
	Type() string
}
