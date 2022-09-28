package start

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/ipfs/go-cid"
	log "github.com/sirupsen/logrus"
)

// DummyTarget is here to detect peers that sniff DHT traffic. No one should ever request data provided by this target.
type DummyTarget struct{}

func NewDummyTarget() *DummyTarget {
	return &DummyTarget{}
}

var _ Target = (*DummyTarget)(nil)

func (dt *DummyTarget) Operation(ctx context.Context, c cid.Cid) error {
	time.Sleep(time.Minute)
	return nil
}

func (dt *DummyTarget) Backoff(ctx context.Context) backoff.BackOff {
	return backoff.WithContext(backoff.NewExponentialBackOff(), ctx)
}

func (dt *DummyTarget) Rate() time.Duration {
	return time.Minute
}

func (dt *DummyTarget) Timeout() time.Duration {
	return 5 * time.Minute
}

func (dt *DummyTarget) Name() string {
	return "dummy"
}

func (dt *DummyTarget) Type() string {
	return "honeypot"
}

func (dt *DummyTarget) CleanUp(ctx context.Context, c cid.Cid) error {
	return nil
}

func (dt *DummyTarget) logEntry() *log.Entry {
	return log.WithField("type", dt.Type()).WithField("name", dt.Name())
}
