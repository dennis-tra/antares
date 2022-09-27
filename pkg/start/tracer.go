package start

import (
	"sync"

	"github.com/ipfs/go-bitswap/message"
	"github.com/ipfs/go-bitswap/tracer"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
	log "github.com/sirupsen/logrus"
)

type Tracer struct {
	cidsLk sync.RWMutex
	cids   map[string]chan<- peer.ID
}

var _ tracer.Tracer = (*Tracer)(nil)

func NewTracer() *Tracer {
	return &Tracer{
		cidsLk: sync.RWMutex{},
		cids:   map[string]chan<- peer.ID{},
	}
}

func (t *Tracer) Register(contentID cid.Cid) <-chan peer.ID {
	log.WithField("cid", contentID).Debugln("Tracer registered CID")

	t.cidsLk.Lock()
	defer t.cidsLk.Unlock()

	ch := make(chan peer.ID)
	t.cids[string(contentID.Bytes())] = ch

	return ch
}

func (t *Tracer) Unregister(contentID cid.Cid) {
	t.cidsLk.Lock()
	defer t.cidsLk.Unlock()
	log.WithField("cid", contentID).Debugln("Tracer unregistered CID")

	ch, ok := t.cids[string(contentID.Bytes())]
	if !ok {
		return
	}

	close(ch)
	delete(t.cids, string(contentID.Bytes()))
}

func (t *Tracer) MessageReceived(id peer.ID, msg message.BitSwapMessage) {
	log.WithField("peerID", id).WithField("size", msg.Size()).Traceln("Received Bitswap message")

	t.cidsLk.RLock()
	defer t.cidsLk.RUnlock()

	for _, e := range msg.Wantlist() {
		ch, ok := t.cids[string(e.Cid.Bytes())]
		if !ok {
			continue
		}

		select {
		case ch <- id:
			log.WithField("peerID", id).WithField("cid", e.Cid).Traceln("Tracer delivered matched message")
		default:
			log.WithField("peerID", id).WithField("cid", e.Cid).Traceln("Tracer dropped matched message")
		}
	}
}

func (t *Tracer) MessageSent(id peer.ID, msg message.BitSwapMessage) {
	log.WithField("peerID", id).WithField("size", msg.Size()).Traceln("Sent Bitswap message")
}
