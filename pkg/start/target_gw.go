package start

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/dennis-tra/antares/pkg/utils"
)

const GatewayURLReplaceStr = "{cid}"

var GatewayTargets = map[string]string{
	"ipfs.io":                 fmt.Sprintf("https://ipfs.io/ipfs/%s", GatewayURLReplaceStr),
	"dweb.link":               fmt.Sprintf("https://dweb.link/ipfs/%s", GatewayURLReplaceStr),
	"gateway.ipfs.io":         fmt.Sprintf("https://gateway.ipfs.io/ipfs/%s", GatewayURLReplaceStr),
	"ipfs.eth.aragon.network": fmt.Sprintf("https://ipfs.eth.aragon.network/ipfs/%s", GatewayURLReplaceStr),
}

type Gateway struct {
	name   string
	urlFmt string
}

func NewGatewayTarget(name string, urlFmt string) *Gateway {
	return &Gateway{
		name:   name,
		urlFmt: urlFmt,
	}
}

var _ Target = (*Gateway)(nil)

func (g *Gateway) Operation(ctx context.Context, c cid.Cid) backoff.Operation {
	logEntry := g.logEntry().WithField("cid", c)
	return func() error {
		u := strings.ReplaceAll(g.urlFmt, GatewayURLReplaceStr, c.String())

		logEntry.WithField("url", u).Infoln("Requesting cid from Gateway")
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return errors.Wrap(err, "new request")
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return errors.Wrap(err, "request do")
		}
		defer resp.Body.Close()

		if !utils.IsSuccessStatusCode(resp) {
			return fmt.Errorf("status code %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "read request body")
		}

		var p Payload
		if err = json.Unmarshal(body, &p); err != nil {
			return errors.Wrap(err, "read request data")
		}

		logEntry.WithField("msg", p.Message).WithField("ts", p.Timestamp).Debugln("Fetched data")

		return nil
	}
}

func (g *Gateway) Backoff(ctx context.Context) backoff.BackOff {
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

func (g *Gateway) Rate() time.Duration {
	return time.Minute
}

func (g *Gateway) Timeout() time.Duration {
	return 11 * time.Minute
}

func (g *Gateway) Name() string {
	return g.name
}

func (g *Gateway) Type() string {
	return "gateway"
}

func (g *Gateway) CleanUp(c cid.Cid) backoff.Operation {
	return func() error { return nil }
}

func (g *Gateway) logEntry() *log.Entry {
	return log.WithField("type", g.Type()).WithField("name", g.Name())
}
