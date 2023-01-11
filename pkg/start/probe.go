package start

import (
	"context"
	"github.com/cenkalti/backoff/v4"
	"github.com/ipfs/go-cid"
	log "github.com/sirupsen/logrus"
)

type Probe interface {
	run(ctx context.Context)
	logEntry() *log.Entry
	wait()
}

func backoffWrap(ctx context.Context, c cid.Cid, fn func(context.Context, cid.Cid) error) backoff.Operation {
	return func() error {
		return fn(ctx, c)
	}
}
