package start

import (
	"context"
	"database/sql"
	"sort"
	"time"

	"go.opencensus.io/tag"

	"github.com/dennis-tra/antares/pkg/metrics"
	"go.opencensus.io/stats"

	"github.com/dennis-tra/antares/pkg/utils"

	"github.com/amit7itz/goset"
	"github.com/cenkalti/backoff/v4"
	blocks "github.com/ipfs/go-block-format"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"

	"github.com/dennis-tra/antares/pkg/config"
	"github.com/dennis-tra/antares/pkg/db"
	"github.com/dennis-tra/antares/pkg/maxmind"
	"github.com/dennis-tra/antares/pkg/models"
)

type ProviderProbe struct {
	host       host.Host
	dbc        *db.Client
	mmc        *maxmind.Client
	config     *config.Config
	dht        *kaddht.IpfsDHT
	bstore     blockstore.Blockstore
	tracer     *Tracer
	target     Target
	probeCount int64
	trackCount int64
	done       chan struct{}
}

func (p *ProviderProbe) run(ctx context.Context) {
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

func (p *ProviderProbe) probeTarget(ctx context.Context) error {
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
			logEntry.Infoln("ProviderProbe operation failed")
			cancel()
		}
	}()
	defer func() {
		op := backoffWrap(tCtx, block.Cid(), p.target.CleanUp)
		bo := p.target.Backoff(ctx)

		if err := backoff.Retry(op, bo); err != nil && !utils.IsContextErr(err) {
			logEntry.WithError(err).Warnln("Error cleaning up resources")
		}
	}()

	select {
	case peerID := <-chPeerID:
		logEntry.WithField("peerID", peerID).Infoln("Tracking peer that requested cid")
		return p.trackPeer(ctx, peerID)
	case <-tCtx.Done():
		return nil
	}
}

func (p *ProviderProbe) notify(err error, dur time.Duration) {
	p.logEntry().WithError(err).WithField("dur", dur).Debugln("Probe operation failed")
}

func (p *ProviderProbe) logEntry() *log.Entry {
	return log.WithField("type", p.target.Type()).WithField("name", p.target.Name())
}

func (p *ProviderProbe) generateContent(ctx context.Context) (*blocks.BasicBlock, func(), error) {
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

func (p *ProviderProbe) trackPeer(ctx context.Context, peerID peer.ID) error {
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

	if p.config.Database.DryRun {
		p.logEntry().Infoln("Skipping database interaction due to --dry-run flag")

		p.logEntry().Infoln("Tracked the following peer:")
		p.logEntry().Infoln("  PeerID", peerID.String())
		p.logEntry().Infoln("  AgentVersion", agentVersion)
		p.logEntry().Infoln("  Protocols", protocols)
		for i, protocol := range protocols {
			p.logEntry().Infof("    [%d] %s\n", i, protocol)
		}
		p.logEntry().Infoln("  MultiAddresses", maddrStrs)
		for i, maddrStr := range maddrStrs {
			p.logEntry().Infof("    [%d] %s\n", i, maddrStr)
		}
		p.logEntry().Infoln("  IPAddresses", ipAddresses)
		for i, ipAddress := range ipAddresses {
			p.logEntry().Infof("    [%d] %s\n", i, ipAddress)
		}
		p.logEntry().Infoln("  Countries", countries)
		p.logEntry().Infoln("  Continents", continents)
		p.logEntry().Infoln("  ASNs", asns)
		p.logEntry().Infoln("  TargetType", p.target.Type())
		p.logEntry().Infoln("  TargetName", p.target.Name())

		return nil
	}

	txn, err := p.dbc.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "begin txn")
	}
	defer func() {
		if err = txn.Rollback(); err != nil && err != sql.ErrTxDone {
			log.WithError(err).Warnln("Error rolling back transaction")
		}
	}()

	dbPeer, err := models.Peers(qm.Expr(
		models.PeerWhere.MultiHash.EQ(peerID.String()),
		models.PeerWhere.TargetName.EQ(p.target.Name()),
	)).One(ctx, txn)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return errors.Wrap(err, "query peer from db")
	}

	if dbPeer == nil {
		dbPeer = &models.Peer{
			MultiHash:      peerID.String(),
			AgentVersion:   null.NewString(agentVersion, agentVersion != ""),
			Protocols:      protocols,
			MultiAddresses: maddrStrs,
			IPAddresses:    ipAddresses,
			Countries:      countries,
			Continents:     continents,
			Asns:           asns,
			TargetType:     p.target.Type(),
			TargetName:     p.target.Name(),
			LastSeenAt:     time.Now(),
		}
		if err = dbPeer.Insert(ctx, txn, boil.Infer()); err != nil {
			return errors.Wrap(err, "insert db peer")
		}
	} else {
		if agentVersion != "" {
			dbPeer.AgentVersion = null.StringFrom(agentVersion)
		}
		if len(protocols) != 0 {
			dbPeer.Protocols = protocols
		}
		if len(maddrStrs) != 0 {
			dbPeer.MultiAddresses = maddrStrs
		}
		if len(ipAddresses) != 0 {
			dbPeer.IPAddresses = ipAddresses
		}
		if len(countries) != 0 {
			dbPeer.Countries = countries
		}
		if len(continents) != 0 {
			dbPeer.Continents = continents
		}
		if len(asns) != 0 {
			dbPeer.Asns = asns
		}
		dbPeer.LastSeenAt = time.Now()
		if _, err = dbPeer.Update(ctx, txn, boil.Infer()); err != nil {
			return errors.Wrap(err, "insert db peer")
		}
	}

	if err = txn.Commit(); err != nil {
		return errors.Wrap(err, "commit txn")
	}

	return nil
}

func (p *ProviderProbe) wait() {
	<-p.done
}
