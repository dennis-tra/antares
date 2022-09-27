package start

import (
	"context"
	"time"

	"github.com/dennis-tra/antares/pkg/maxmind"

	"github.com/cenkalti/backoff/v4"
	"github.com/ipfs/go-bitswap"
	bsnet "github.com/ipfs/go-bitswap/network"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	"github.com/libp2p/go-libp2p"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/routing"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/dennis-tra/antares/pkg/config"
	"github.com/dennis-tra/antares/pkg/db"
)

// The Scheduler TODO
type Scheduler struct {
	// The libp2p node that's used to TODO
	host host.Host

	// The database client
	dbc *db.Client

	// TODO
	mmc *maxmind.Client

	// The configuration of timeouts etc.
	config *config.Config

	// TODO
	dht *kaddht.IpfsDHT

	// TODO
	tracer *Tracer

	// TODO
	bitswap *bitswap.Bitswap

	// TODO
	bstore blockstore.Blockstore

	// TODO
	targets []Target
}

type Target interface {
	Backoff(ctx context.Context) backoff.BackOff
	Operation(ctx context.Context, c cid.Cid) backoff.Operation
	Timeout() time.Duration
	Rate() time.Duration
	Name() string
	Type() string
	CleanUp(c cid.Cid)
}

// NewScheduler TODO
func NewScheduler(ctx context.Context, conf *config.Config, dbc *db.Client, mmc *maxmind.Client) (*Scheduler, error) {
	mgr, err := rcmgr.NewResourceManager(rcmgr.NewFixedLimiter(rcmgr.InfiniteLimits))
	if err != nil {
		return nil, errors.Wrap(err, "new resource manager")
	}

	var dht *kaddht.IpfsDHT
	h, err := libp2p.New(
		libp2p.Identity(conf.PrivKey),
		libp2p.ListenAddrs(conf.ListenAddrTCP, conf.ListenAddrQUIC),
		libp2p.UserAgent("antares/"+conf.Version),
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			dht, err = kaddht.New(ctx, h)
			return dht, err
		}),
		libp2p.ResourceManager(mgr),
	)
	if err != nil {
		return nil, errors.Wrap(err, "new libp2p host")
	}

	t := NewTracer()

	network := bsnet.NewFromIpfsHost(h, dht)
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	bstore := blockstore.NewBlockstore(ds)
	bs := bitswap.New(ctx, network, bstore, bitswap.WithTracer(t))

	targets := []Target{}
	for name, url := range GatewayTargets {
		targets = append(targets, NewGatewayTarget(name, url))
	}

	targets = append(targets, NewPinata(h, conf.Authorizations["pinata"]))

	s := &Scheduler{
		host:    h,
		dbc:     dbc,
		mmc:     mmc,
		config:  conf,
		dht:     dht,
		tracer:  t,
		bitswap: bs,
		bstore:  bstore,
		targets: targets,
	}

	return s, nil
}

func (s *Scheduler) StartProbes(ctx context.Context) error {
	for _, bp := range kaddht.GetDefaultBootstrapPeerAddrInfos() {
		log.WithField("peerID", bp.ID).Infoln("Connecting to bootstrap peer")
		if err := s.host.Connect(ctx, bp); err != nil {
			return errors.Wrap(err, "connect to bootstrap peer")
		}
	}

	var probes []*Probe
	for _, target := range s.targets {
		log.Infof("Starting %s probe %s...", target.Type(), target.Name())
		p := s.newProbe(target)
		probes = append(probes, p)
		go p.run(ctx)
	}

	log.WithField("count", len(s.targets)).Infoln("Initialized all target probes!")
	<-ctx.Done()

	for _, p := range probes {
		log.WithField("type", p.target.Type()).WithField("name", p.target.Name()).Infoln("Waiting for probe to stop")
		<-p.done
	}

	return nil
}

func (s *Scheduler) newProbe(target Target) *Probe {
	return &Probe{
		host:   s.host,
		dbc:    s.dbc,
		mmc:    s.mmc,
		config: s.config,
		dht:    s.dht,
		bstore: s.bstore,
		tracer: s.tracer,
		target: target,
		done:   make(chan struct{}),
	}
}
