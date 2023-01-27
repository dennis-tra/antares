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

// Payload is the underlying data that gets announced to the network
type Payload struct {
	Message   string
	Timestamp time.Time
	Random    []byte
	Signature []byte
}

// NewPayload generates 100 bytes of random data and initializes a Payload
// data structure. It's also signing the data for no reason.
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

// Bytes returns the json representation of the data embedded into a
// DAG node so that IPFS can make sense of the data.
func (p *Payload) Bytes() ([]byte, error) {
	dat, err := p.JsonBytes()
	if err != nil {
		return nil, errors.Wrap(err, "new probe data")
	}

	return dag.NodeWithData(ft.FilePBData(dat, uint64(len(dat)))).Marshal()
}

// JsonBytes returns the json representation of the data.
func (p *Payload) JsonBytes() ([]byte, error) {
	return json.Marshal(p)
}
