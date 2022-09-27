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

	"github.com/dennis-tra/antares/pkg/config"
	"github.com/dennis-tra/antares/pkg/utils"
)

const InfuraTargetName = "infura"

type Infura struct {
	h        host.Host
	username string
	password string
}

func NewInfura(h host.Host, conf *config.Config) (*Infura, error) {
	parts := strings.Split(conf.Authorizations[InfuraTargetName], ",")
	if len(parts) != 2 {
		return nil, fmt.Errorf("malformed infura credentials")
	}

	return &Infura{
		h:        h,
		username: parts[0],
		password: parts[1],
	}, nil
}

var _ Target = (*Infura)(nil)

func (i *Infura) Operation(ctx context.Context, c cid.Cid) backoff.Operation {
	logEntry := i.logEntry().WithField("cid", c)
	return func() error {
		logEntry.Infoln("Pinning cid to Infura...")
		req, err := http.NewRequest(http.MethodPost, "https://ipfs.infura.io:5001/api/v0/pin/add?arg=/ipfs/"+c.String(), nil)
		if err != nil {
			return errors.Wrap(err, "new infura http request")
		}
		req.SetBasicAuth(i.username, i.password)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return errors.Wrap(err, "pin file to infura")
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "read request body")
		}
		logEntry.Debugln(string(respBody))

		if !utils.IsSuccessStatusCode(resp) {
			return fmt.Errorf("status code %d", resp.StatusCode)
		}

		return nil
	}
}

func (i *Infura) Backoff(ctx context.Context) backoff.BackOff {
	bo := &backoff.ExponentialBackOff{
		InitialInterval:     5 * time.Second,
		RandomizationFactor: 0.5,
		Multiplier:          1.5,
		MaxInterval:         2 * time.Minute,
		MaxElapsedTime:      10 * time.Minute,
		Stop:                backoff.Stop,
		Clock:               backoff.SystemClock,
	}
	return backoff.WithContext(bo, ctx)
}

func (i *Infura) Rate() time.Duration {
	return time.Minute
}

func (i *Infura) Timeout() time.Duration {
	return 15 * time.Minute
}

func (i *Infura) Name() string {
	return InfuraTargetName
}

func (i *Infura) Type() string {
	return "pinning-service"
}

func (i *Infura) CleanUp(c cid.Cid) backoff.Operation {
	logEntry := i.logEntry().WithField("cid", c)

	return func() error {
		req, err := http.NewRequest(http.MethodPost, "https://ipfs.infura.io:5001/api/v0/pin/rm?arg=/ipfs/"+c.String(), nil)
		if err != nil {
			return errors.Wrap(err, "new infura http request")
		}
		req.SetBasicAuth(i.username, i.password)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			logEntry.WithError(err).Warnln("Error unpinning cid from infura")
			return errors.Wrap(err, "infura delete request")
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			logEntry.WithError(err).Warnln("read all")
			return errors.Wrap(err, "reading infura delete response body")
		}
		logEntry.Debugln(string(respBody))

		if !utils.IsSuccessStatusCode(resp) {
			logEntry.WithField("status", resp.StatusCode).Warnln("Error unpinning cid from infura")
			return errors.Wrap(err, "infura delete non-success status code")
		}

		return nil
	}
}

func (i *Infura) logEntry() *log.Entry {
	return log.WithField("type", i.Type()).WithField("name", i.Name())
}
