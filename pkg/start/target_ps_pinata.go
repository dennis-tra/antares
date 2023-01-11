package start

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/host"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/dennis-tra/antares/pkg/utils"
)

const PinataTargetName = "pinata"

type Pinata struct {
	h    host.Host
	auth string
}

func NewPinata(h host.Host, auth string) (PinTarget, error) {
	return &Pinata{h: h, auth: auth}, nil
}

var _ Target = (*Pinata)(nil)

func (p *Pinata) Operation(ctx context.Context, c cid.Cid) error {
	logEntry := p.logEntry().WithField("cid", c)
	logEntry.Infoln("Pinning cid to Pinata...")

	var publicMaddr ma.Multiaddr
	for _, maddr := range p.h.Addrs() {
		if manet.IsPublicAddr(maddr) {
			publicMaddr = maddr
			break
		}
	}
	var popts *PinataOptions
	if publicMaddr != nil {
		popts = &PinataOptions{HostNodes: []string{
			publicMaddr.String() + "/p2p/" + p.h.ID().String(),
		}}
	}

	payload := PinataRequest{
		HashToPin: c.String(),
		PinataMetadata: &PinataMetadata{
			Name: "Antares " + time.Now().String(),
		},
		PinataOptions: popts,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "marshal request payload")
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.pinata.cloud/pinning/pinByHash", bytes.NewBuffer(data))
	if err != nil {
		return errors.Wrap(err, "new request")
	}
	req.Header.Add("Authorization", "Bearer "+p.auth)
	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "pin file to pinata")
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

func (p *Pinata) Backoff(ctx context.Context) backoff.BackOff {
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

func (p *Pinata) Rate() time.Duration {
	return 5 * time.Minute
}

func (p *Pinata) Timeout() time.Duration {
	return 10 * time.Minute
}

func (p *Pinata) Name() string {
	return "pinata"
}

func (p *Pinata) Type() string {
	return "pinning-service"
}

func (p *Pinata) CleanUp(ctx context.Context, c cid.Cid) error {
	logEntry := p.logEntry().WithField("cid", c)

	logEntry.Infoln("Unpinning cid from Pinata...")

	req, err := http.NewRequest(http.MethodDelete, "https://api.pinata.cloud/pinning/unpin/"+c.String(), nil)
	if err != nil {
		return errors.Wrap(err, "new request")
	}
	req.Header.Add("Authorization", "Bearer "+p.auth)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "unpin cid from pinata")
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

func (p *Pinata) logEntry() *log.Entry {
	return log.WithField("type", p.Type()).WithField("name", p.Name())
}
