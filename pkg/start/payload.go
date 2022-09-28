package start

import (
	"crypto/rand"
	"encoding/json"
	"time"

	dag "github.com/ipfs/go-merkledag"
	ft "github.com/ipfs/go-unixfs"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/pkg/errors"
)

type Payload struct {
	Message   string
	Timestamp time.Time
	Random    []byte
	Signature []byte
}

func NewPayload(key crypto.PrivKey) (*Payload, error) {
	buf := make([]byte, 100)
	_, err := rand.Read(buf)
	if err != nil {
		return nil, errors.Wrap(err, "read random data")
	}

	p := &Payload{
		Message:   "Antares Test Data",
		Timestamp: time.Now(),
		Random:    buf,
		Signature: nil,
	}

	dat, err := json.Marshal(p)
	if err != nil {
		return nil, errors.Wrap(err, "marshal probe data")
	}

	signature, err := key.Sign(dat)
	if err != nil {
		return nil, errors.Wrap(err, "signing probe data")
	}

	p.Signature = signature

	return p, nil
}

func (p *Payload) Bytes() ([]byte, error) {
	dat, err := json.Marshal(p)
	if err != nil {
		return nil, errors.Wrap(err, "new probe data")
	}

	return dag.NodeWithData(ft.FilePBData(dat, uint64(len(dat)))).Marshal()
}
