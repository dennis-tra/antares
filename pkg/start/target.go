package start

import (
	"context"
	blocks "github.com/ipfs/go-block-format"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/host"
)

type PinningServiceTargetConstructor = func(host host.Host, auth string) (PinTarget, error)

var PinningServiceTargetConstructors = map[string]PinningServiceTargetConstructor{
	InfuraTargetName: NewInfura,
	PinataTargetName: NewPinata,
}

type UploadServiceTargetConstructor = func(host host.Host, auth string) (UploadTarget, error)

var UploadServiceTargetConstructors = map[string]UploadServiceTargetConstructor{
	Web3TargetName: NewWeb3,
}

type Target interface {
	Backoff(ctx context.Context) backoff.BackOff
	Timeout() time.Duration
	Rate() time.Duration
	Name() string
	Type() string
}
type PinTarget interface {
	Target
	Operation(ctx context.Context, c cid.Cid) error
}

type CleanupTarget interface {
	Target
	CleanUp(ctx context.Context, c cid.Cid) error
}

type UploadTarget interface {
	Target
	UploadContent(ctx context.Context, block *blocks.BasicBlock) error
}
