package start

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/dennis-tra/antares/pkg/utils"
)

const InfuraTargetName = "infura"

type Infura struct {
	username string
	password string
}

func NewInfura(h host.Host, auth string) (PinTarget, error) {
	parts := strings.Split(auth, ",")
	if len(parts) != 2 {
		return nil, fmt.Errorf("malformed infura credentials")
	}

	return &Infura{
		username: parts[0],
		password: parts[1],
	}, nil
}

var _ Target = (*Infura)(nil)

func (i *Infura) Operation(ctx context.Context, c cid.Cid) error {
	logEntry := i.logEntry().WithField("cid", c)
	logEntry.Infoln("Pinning cid to Infura...")
	req, err := http.NewRequest(http.MethodPost, "https://ipfs.infura.io:5001/api/v0/pin/add?arg=/ipfs/"+c.String(), nil)
	if err != nil {
		return errors.Wrap(err, "new request")
	}
	req.SetBasicAuth(i.username, i.password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "pin file to infura")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "read response body")
	}
	logEntry.Debugln("Pin response:", string(respBody))

	if !utils.IsSuccessStatusCode(resp) {
		return fmt.Errorf("status code %d", resp.StatusCode)
	}

	return nil
}

func (i *Infura) Backoff(ctx context.Context) backoff.BackOff {
	bo := &backoff.ExponentialBackOff{
		InitialInterval:     time.Minute,
		RandomizationFactor: 0.5,
		Multiplier:          1.2,
		MaxInterval:         5 * time.Minute,
		MaxElapsedTime:      10 * time.Minute,
		Stop:                backoff.Stop,
		Clock:               backoff.SystemClock,
	}
	return backoff.WithContext(bo, ctx)
}

func (i *Infura) Rate() time.Duration {
	return 5 * time.Minute
}

func (i *Infura) Timeout() time.Duration {
	return 10 * time.Minute
}

func (i *Infura) Name() string {
	return InfuraTargetName
}

func (i *Infura) Type() string {
	return "pinning-service"
}

func (i *Infura) CleanUp(ctx context.Context, c cid.Cid) error {
	logEntry := i.logEntry().WithField("cid", c)
	logEntry.Debugln("Unpinning cid from Infura...")

	req, err := http.NewRequest(http.MethodPost, "https://ipfs.infura.io:5001/api/v0/pin/rm?arg=/ipfs/"+c.String(), nil)
	if err != nil {
		return errors.Wrap(err, "new infura http request")
	}
	req.SetBasicAuth(i.username, i.password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "unpin cid from infura")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "read response body")
	}
	logEntry.Debugln("Unpin response:", string(respBody))

	if !utils.IsSuccessStatusCode(resp) {
		return fmt.Errorf("status code %d", resp.StatusCode)
	}

	return nil
}

func (i *Infura) logEntry() *log.Entry {
	return log.WithField("type", i.Type()).WithField("name", i.Name())
}
