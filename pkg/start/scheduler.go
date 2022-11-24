package start

import (
	"context"

	"github.com/ipfs/go-bitswap"
	bsnet "github.com/ipfs/go-bitswap/network"
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
	"github.com/dennis-tra/antares/pkg/maxmind"
)

// The Scheduler is responsible for the initialization of Targets and Probes. Targets are entities like gateways
// or pinning services. Probes can be configured with a specific Target and carry out the publication of content and
// later the request through the Target. After all targets are initialized from the configuration, they get assigned
// a Probe. These probes are then instructed to start doing their thing - which means, announcing CIDs to the DHT and
// then requesting it through the associated target.
type Scheduler struct {
	// The libp2p node that's used to announce CIDs to the DHT and handle the Bitswap exchange of the data. The Bitswap
	// traffic is then used to detect who requested those CIDs.
	host host.Host

	// A handle on the database to issue queries.
	dbc *db.Client

	// A handle on the Maxmind GeoIP2 database to resolve IP addresses to country and continent information.
	mmc *maxmind.Client

	// A reference to the configuration of Antares
	config *config.Config

	// A reference to the Kademlia DHT to be able to provide CIDs to the network.
	dht *kaddht.IpfsDHT

	// The tracer is handed into the Bitswap exchange submodule and implements two methods that get called whenever
	// a Bitswap message leaves the Antares libp2p host or is received by it.
	tracer *Tracer

	// A reference to the underlying blockstore that Bitswap uses to deliver the blocks that were previously advertised
	// via their CID to the DHT
	bstore blockstore.Blockstore

	// A list of Targets to probe.
	targets []Target
}

// NewScheduler initializes a new libp2p host with the given configuration handles to a persistent storage and Maxmind
// GeoIP2 database.
func NewScheduler(ctx context.Context, conf *config.Config, dbc *db.Client, mmc *maxmind.Client) (*Scheduler, error) {
	// TODO: Still haven't fully grasped how to properly configure the resource manager...
	mgr, err := rcmgr.NewResourceManager(rcmgr.NewFixedLimiter(rcmgr.InfiniteLimits))
	if err != nil {
		return nil, errors.Wrap(err, "new resource manager")
	}

	// Initialize the libp2p host
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

	// Create a new tracer
	t := NewTracer()

	// Configure the Bitswap submodule
	network := bsnet.NewFromIpfsHost(h, dht)
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	bstore := blockstore.NewBlockstore(ds)

	// Register the bitswap protocol handler and start handling new messages, connectivity events, etc.
	// This is the point were we hand in the tracer to be in the loop of what's going on.
	bitswap.New(ctx, network, bstore, bitswap.WithTracer(t))

	// Initialize all configured targets
	targets, err := initTargets(h, conf)
	if err != nil {
		return nil, errors.Wrap(err, "init targets")
	}

	return &Scheduler{
		host:    h,
		dbc:     dbc,
		mmc:     mmc,
		config:  conf,
		dht:     dht,
		tracer:  t,
		bstore:  bstore,
		targets: targets,
	}, nil
}

// initTargets takes the current configuration options and constructs corresponding target data structures.
// A Target is just the entity that we are probing to detect their PeerIDs and can be gateways or pinning services.
// It always adds a dummy target. For each entry in the `Gateways` and `PinningServices` list it also
// creates a corresponding target.
func initTargets(h host.Host, conf *config.Config) ([]Target, error) {
	// Always add the dummy target to detect peers that are proactively
	targets := []Target{NewDummyTarget()}

	// Add all configured gateways
	for _, gw := range conf.Gateways {
		targets = append(targets, NewGatewayTarget(gw.Name, gw.URL))
	}

	// Add all configured pinning services
	for _, ps := range conf.PinningServices {
		tc, found := PinningServiceTargetConstructors[ps.Target]
		if !found {
			log.Warnf("no pinning service constructor for target %s\n", ps.Target)
			continue
		}

		pst, err := tc(h, ps.Authorization)
		if err != nil {
			return nil, errors.Wrapf(err, "constructing pinning service target: %s", ps.Target)
		}

		targets = append(targets, pst)
	}

	// Add all configured upload services
	for _, ps := range conf.UploadServices {
		tc, found := UploadServiceTargetConstructors[ps.Target]
		if !found {
			log.Warnf("no upload service constructor for target %s\n", ps.Target)
			continue
		}

		pst, err := tc(h, ps.Authorization)
		if err != nil {
			return nil, errors.Wrapf(err, "constructing upload service target: %s", ps.Target)
		}

		targets = append(targets, pst)
	}

	return targets, nil
}

// StartProbes connects to the IPFS bootstrap peers and starts each target probe in their own go-routine.
func (s *Scheduler) StartProbes(ctx context.Context) error {
	// Connect to IPFS bootstrap peers
	for _, bp := range kaddht.GetDefaultBootstrapPeerAddrInfos() {
		log.WithField("peerID", bp.ID).Infoln("Connecting to bootstrap peer")
		if err := s.host.Connect(ctx, bp); err != nil {
			return errors.Wrap(err, "connect to bootstrap peer")
		}
	}

	// Start all probes
	var probes []*Probe
	for _, target := range s.targets {
		log.Infof("Starting %s probe %s...", target.Type(), target.Name())
		p := s.newProbe(target)
		probes = append(probes, p)
		go p.run(ctx)
	}

	// Block until the user wants to stop
	log.WithField("count", len(s.targets)).Infoln("Initialized all target probes!")
	<-ctx.Done()

	// The user wanted to stop the program, wait until all probes have gracefully stopped
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
