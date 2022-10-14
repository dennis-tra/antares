package start

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/cenkalti/backoff/v4"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"time"
)

const Web3TargetName = "web3"

type Web3 struct {
	h    host.Host
	auth string
}

func NewWeb3(h host.Host, auth string) (Target, error) {
	return &Web3{h: h, auth: auth}, nil
}

var _ ContentProvidingTarget = (*Web3)(nil)

func (t *Web3) Operation(ctx context.Context, c cid.Cid) error {
	return nil
}

func (t *Web3) Backoff(ctx context.Context) backoff.BackOff {
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

func (t *Web3) Rate() time.Duration {
	return 5 * time.Minute
}

func (t *Web3) Timeout() time.Duration {
	return 10 * time.Minute
}

func (t *Web3) Name() string {
	return "web3"
}

func (t *Web3) Type() string {
	return "upload service"
}

func (t *Web3) CleanUp(ctx context.Context, c cid.Cid) error {
	return nil
}

func (t *Web3) GenerateContent(ctx context.Context, privKey crypto.PrivKey) (*cid.Cid, func(), error) {
	logEntry := t.logEntry()
	pl, err := NewPayload(privKey)
	if err != nil {
		return nil, nil, errors.Wrap(err, "new payload data")
	}

	data, err := pl.Bytes()
	if err != nil {
		return nil, nil, errors.Wrap(err, "payload bytes")
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.web3.storage/upload", bytes.NewBuffer(data))
	if err != nil {
		return nil, nil, errors.Wrap(err, "new request")
	}
	req.Header.Add("Authorization", "Bearer "+t.auth)
	req.Header.Add("Content-Type", "application/octet-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, errors.Wrap(err, "upload file to web3.storage")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, errors.Wrap(err, "read response body")
	}
	logEntry.Debugln("Upload response: ", string(respBody))

	var jsonResp Web3UploadResponse
	json.Unmarshal(respBody, &jsonResp)

	cid, err := cid.Parse(jsonResp.Cid)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Parse cid")
	}

	delFunc := func() {
		logEntry.Infoln("Removing content from web3")
		// TODO: find an api endpoint that allowes delete
		logEntry.Errorf("web3.storage api does not allow delete, please delete the item with the cid: %s", cid.String())
	}

	return &cid, delFunc, nil
}

func (t *Web3) logEntry() *log.Entry {
	return log.WithField("type", t.Type()).WithField("name", t.Name())
}

type Web3UploadResponse struct {
	Cid    string `json:"cid"`
	CarCid string `json:"carCid"`
}
