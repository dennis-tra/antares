package start

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"

	"github.com/libp2p/go-libp2p/core/host"

	"github.com/cenkalti/backoff/v4"
	"github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type Pinata struct {
	h    host.Host
	auth string
}

type PinataRequest struct {
	HashToPin      string          `json:"hashToPin"`
	PinataMetadata *PinataMetadata `json:"pinataMetadata,omitempty"`
	PinataOptions  *PinataOptions  `json:"pinataOptions,omitempty"`
}

type PinataMetadata struct {
	Name string `json:"name"`
}

type PinataOptions struct {
	HostNodes []string `json:"hostNodes"`
}

func NewPinata(h host.Host, auth string) *Pinata {
	return &Pinata{
		h:    h,
		auth: auth,
	}
}

var _ Target = (*Pinata)(nil)

func (p *Pinata) Operation(ctx context.Context, c cid.Cid) backoff.Operation {
	logEntry := p.logEntry().WithField("cid", c)
	return func() error {
		var publicMaddr ma.Multiaddr
		for _, maddr := range p.h.Addrs() {
			if manet.IsPublicAddr(maddr) {
				publicMaddr = maddr
				break
			}
		}
		var popts *PinataOptions
		if publicMaddr != nil {
			popts = &PinataOptions{HostNodes: []string{publicMaddr.String()}}
		}

		payload := PinataRequest{
			HashToPin: c.String(),
			PinataMetadata: &PinataMetadata{
				Name: "Antares",
			},
			PinataOptions: popts,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return errors.Wrap(err, "marshal pinata request payload")
		}

		logEntry.Infoln("Pinning cid to Pinata...")
		req, err := http.NewRequest(http.MethodPost, "https://api.pinata.cloud/pinning/pinByHash", bytes.NewBuffer(data))
		if err != nil {
			return errors.Wrap(err, "new pinata http request")
		}
		req.Header.Add("Authorization", "Bearer "+p.auth)
		req.Header.Add("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return errors.Wrap(err, "pin file to pinata")
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return fmt.Errorf("status code %d", resp.StatusCode)
		}

		if _, err = io.ReadAll(resp.Body); err != nil {
			return errors.Wrap(err, "read request body")
		}

		return nil
	}
}

func (p *Pinata) Backoff(ctx context.Context) backoff.BackOff {
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

func (p *Pinata) Rate() time.Duration {
	return time.Minute
}

func (p *Pinata) Timeout() time.Duration {
	return 15 * time.Minute
}

func (p *Pinata) Name() string {
	return "pinata"
}

func (p *Pinata) Type() string {
	return "pinning-service"
}

func (p *Pinata) CleanUp(c cid.Cid) {
	logEntry := p.logEntry().WithField("cid", c)

	req, err := http.NewRequest(http.MethodDelete, "https://api.pinata.cloud/pinning/unpin/"+c.String(), nil)
	if err != nil {
		p.logEntry().WithError(err).WithField("cid", c).Warnln("Error creating request object to unpin cid from pinata")
		return
	}
	req.Header.Add("Authorization", "Bearer "+p.auth)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logEntry.WithError(err).Warnln("Error unpinning cid from pinata")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		logEntry.WithField("status", resp.StatusCode).Warnln("Error unpinning cid from pinata")
		return
	}
}

func (p *Pinata) logEntry() *log.Entry {
	return log.WithField("type", p.Type()).WithField("name", p.Name())
}
