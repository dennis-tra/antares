package start

import (
	"context"
	"sort"
	"time"

	"go.opencensus.io/tag"

	"github.com/dennis-tra/antares/pkg/metrics"
	"go.opencensus.io/stats"

	"github.com/dennis-tra/antares/pkg/utils"

	"github.com/amit7itz/goset"
	"github.com/cenkalti/backoff/v4"
	"github.com/dennis-tra/antares/pkg/config"
	"github.com/dennis-tra/antares/pkg/db"
	"github.com/dennis-tra/antares/pkg/maxmind"
	blocks "github.com/ipfs/go-block-format"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type PinProbe struct {
	host       host.Host
	dbc        *db.Client
	mmc        *maxmind.Client
	config     *config.Config
	dht        *kaddht.IpfsDHT
	bstore     blockstore.Blockstore
	tracer     *Tracer
	target     PinTarget
	probeCount int64
	trackCount int64
	done       chan struct{}
}

func (p *PinProbe) run(ctx context.Context) {
	defer close(p.done)

	ctx, err := tag.New(ctx, tag.Insert(metrics.KeyTargetName, p.target.Name()), tag.Insert(metrics.KeyTargetType, p.target.Type()))
	if err != nil {
		p.logEntry().WithError(err).Errorln("Error creating new tag context")
		return
	}

	// Only request
	throttle := NewThrottle(1, p.target.Rate())
	defer throttle.Stop()

	for {
		// Giving cancelled context precedence
		select {
		case <-ctx.Done():
			return
		default:
		}

		p.logEntry().WithField("rate", p.target.Rate()).Infoln("Checking probe lease...")
		select {
		case <-ctx.Done():
			return
		case <-throttle.C:
		}

		err := p.probeTarget(ctx)
		if utils.IsContextErr(err) {
			return
		} else if err != nil {
			p.logEntry().WithError(err).Warnln("Error probing target")
		}
	}
}

func (p *PinProbe) probeTarget(ctx context.Context) error {
	p.probeCount += 1
	stats.Record(ctx, metrics.ProbeCount.M(p.probeCount))

	block, teardown, err := p.generateContent(ctx)
	defer teardown()
	if err != nil {
		return errors.Wrap(err, "generate content")
	}
	logEntry := p.logEntry().WithField("cid", block.Cid())

	logEntry.Infoln("Registering cid with tracer")
	chPeerID := p.tracer.Register(block.Cid())
	defer p.tracer.Unregister(block.Cid())

	logEntry.Infoln("Providing cid in the dht")
	err = p.dht.Provide(ctx, block.Cid(), true)
	if err != nil {
		return errors.Wrap(err, "dht provide content")
	}

	tCtx, cancel := context.WithTimeout(ctx, p.target.Timeout())
	defer cancel()
	go func() {
		logEntry.Infoln("Starting probe operation")

		op := backoffWrap(tCtx, block.Cid(), p.target.Operation)
		bo := p.target.Backoff(tCtx)

		if err = backoff.RetryNotify(op, bo, p.notify); err != nil && !utils.IsContextErr(err) {
			logEntry.Infoln("Probe operation failed")
			cancel()
		}
	}()
	defer cleanupProbe(tCtx, logEntry, p.target, block.Cid())

	select {
	case peerID := <-chPeerID:
		logEntry.WithField("peerID", peerID).Infoln("Tracking peer that requested cid")
		return p.trackPeer(ctx, peerID)
	case <-tCtx.Done():
		return nil
	}
}

func (p *PinProbe) notify(err error, dur time.Duration) {
	p.logEntry().WithError(err).WithField("dur", dur).Debugln("Probe operation failed")
}

func (p *PinProbe) logEntry() *log.Entry {
	return log.WithField("type", p.target.Type()).WithField("name", p.target.Name())
}

func (p *PinProbe) generateContent(ctx context.Context) (*blocks.BasicBlock, func(), error) {
	pl, err := NewPayload(p.config.PrivKey)
	if err != nil {
		return nil, nil, errors.Wrap(err, "new payload data")
	}

	data, err := pl.Bytes()
	if err != nil {
		return nil, nil, errors.Wrap(err, "payload bytes")
	}

	block := blocks.NewBlock(data)
	logEntry := p.logEntry().WithField("cid", block.Cid())

	logEntry.Infoln("Generated content")
	err = p.bstore.Put(ctx, block)
	if err != nil {
		return nil, nil, errors.Wrap(err, "put block in blockstore")
	}

	return block, func() {
		logEntry.Infoln("Removing content from blockstore")
		if err = p.bstore.DeleteBlock(ctx, block.Cid()); err != nil {
			logEntry.WithError(err).Warnln("Could not delete block")
		}
	}, nil
}

func (p *PinProbe) trackPeer(ctx context.Context, peerID peer.ID) error {
	p.trackCount += 1
	stats.Record(ctx, metrics.TrackCount.M(p.trackCount))

	ps := p.host.Peerstore()

	protocols, err := ps.GetProtocols(peerID)
	if err != nil {
		protocols = nil
	}

	var agentVersion string
	if val, err := ps.Get(peerID, "AgentVersion"); err == nil {
		agentVersion = val.(string)
	}

	maddrSet := map[string]ma.Multiaddr{}
	for _, maddr := range ps.Addrs(peerID) {
		maddrSet[maddr.String()] = maddr
	}
	for _, conn := range p.host.Network().ConnsToPeer(peerID) {
		maddr := conn.RemoteMultiaddr()
		maddrSet[maddr.String()] = maddr
	}

	maddrStrs := []string{}
	ipAddressesSet := goset.NewSet[string]()
	countriesSet := goset.NewSet[string]()
	continentsSet := goset.NewSet[string]()
	asnsSet := goset.NewSet[int64]()

	for maddrStr, maddr := range maddrSet {
		if utils.IsRelayedMaddr(maddr) || !manet.IsPublicAddr(maddr) {
			continue
		}

		maddrStrs = append(maddrStrs, maddrStr)
		maddrInfos, err := p.mmc.MaddrInfo(ctx, maddr)
		if err != nil {
			continue
		}

		for ipAddress, maddrInfo := range maddrInfos {
			ipAddressesSet.Add(ipAddress)
			countriesSet.Add(maddrInfo.Country)
			continentsSet.Add(maddrInfo.Continent)
			asnsSet.Add(int64(maddrInfo.ASN))
		}
	}

	ipAddressesSet.Discard("")
	countriesSet.Discard("")
	continentsSet.Discard("")
	asnsSet.Discard(0)

	ipAddresses := ipAddressesSet.Items()
	countries := countriesSet.Items()
	continents := continentsSet.Items()
	asns := asnsSet.Items()

	sort.Strings(ipAddresses)
	sort.Strings(countries)
	sort.Strings(continents)
	sort.Slice(asns, func(i, j int) bool { return asns[i] < asns[j] })

	return insertModel(ctx, p.config.Database.DryRun, p.dbc, p.logEntry(), protocols, agentVersion, peerID,
		ipAddresses, maddrStrs, countries, continents, asns, p.target.Type(), p.target.Name())
}

func (p *PinProbe) wait() {
	<-p.done
}
