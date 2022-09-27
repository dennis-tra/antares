package start

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/ipfs/go-cid"
)

type Target interface {
	Backoff(ctx context.Context) backoff.BackOff
	Operation(ctx context.Context, c cid.Cid) backoff.Operation
	Timeout() time.Duration
	Rate() time.Duration
	Name() string
	Type() string
	CleanUp(c cid.Cid) backoff.Operation
}
