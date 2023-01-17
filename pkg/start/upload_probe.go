package start

import (
	"context"
	"github.com/amit7itz/goset"
	"github.com/cenkalti/backoff/v4"
	"github.com/dennis-tra/antares/pkg/config"
	"github.com/dennis-tra/antares/pkg/db"
	"github.com/dennis-tra/antares/pkg/maxmind"
	"github.com/dennis-tra/antares/pkg/metrics"
	"github.com/dennis-tra/antares/pkg/utils"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	ipfsutils "github.com/ipfs/go-ipfs-util"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"sort"
	"time"
)

type UploadProbe struct {
	host       host.Host
	dbc        *db.Client
	dht        *kaddht.IpfsDHT
	mmc        *maxmind.Client
	config     *config.Config
	target     UploadTarget
	probeCount int64
	trackCount int64
	done       chan struct{}
}

func (u *UploadProbe) run(ctx context.Context) {
	defer close(u.done)

	ctx, err := tag.New(ctx, tag.Insert(metrics.KeyTargetName, u.target.Name()), tag.Insert(metrics.KeyTargetType, u.target.Type()))
	if err != nil {
		u.logEntry().WithError(err).Errorln("Error creating new tag context")
		return
	}

	// Only request
	throttle := NewThrottle(1, u.target.Rate())
	defer throttle.Stop()

	for {
		// Giving cancelled context precedence
		select {
		case <-ctx.Done():
			return
		default:
		}

		u.logEntry().WithField("rate", u.target.Rate()).Infoln("Checking probe lease...")
		select {
		case <-ctx.Done():
			return
		case <-throttle.C:
		}

		err := u.probeTarget(ctx)
		if utils.IsContextErr(err) {
			return
		} else if err != nil {
			u.logEntry().WithError(err).Warnln("Error probing target")
		}
	}
}

func (u *UploadProbe) probeTarget(ctx context.Context) error {
	u.probeCount += 1
	stats.Record(ctx, metrics.ProbeCount.M(u.probeCount))

	block, err := u.generateContent(ctx)
	if err != nil {
		return errors.Wrap(err, "generate content")
	}
	u.logEntry().Infoln("foo")

	tCtx, cancel := context.WithTimeout(ctx, u.target.Timeout())
	logEntry := u.logEntry().WithField("cid", block.Cid().String())

	defer cancel()

	//go func() {
	logEntry.Infoln("Starting probe operation")

	op := func() error {
		return u.target.UploadContent(tCtx, block)
	}
	bo := u.target.Backoff(tCtx)

	if err = backoff.RetryNotify(op, bo, u.notify); err != nil && !utils.IsContextErr(err) {
		logEntry.Infoln("Probe operation failed")
		cancel()
	}
	//}()
	defer cleanupProbe(tCtx, logEntry, u.target, block.Cid())

	//chProvider := u.dht.FindProvidersAsync(tCtx, block.Cid(), 10)
	addresses, err := u.dht.FindProviders(tCtx, block.Cid())
	if err != nil {
		return errors.Wrap(err, "find providers")
	}

	for _, addr := range addresses {
		logEntry.WithField("addr", addr).Infoln("Found provider")
		u.trackProvider(tCtx, addr)
	}
	return nil

	/*for {
		select {
		case peer := <-chProvider:
			if err := u.trackProvider(ctx, peer); err != nil {
				return err
			}
		case <-tCtx.Done():
			return nil
		}
	}*/
}

func (u *UploadProbe) generateContent(ctx context.Context) (*blocks.BasicBlock, error) {

	payload, err := NewPayload(u.config.PrivKey)
	if err != nil {
		return nil, errors.Wrap(err, "new payload data")
	}
	data, err := payload.JsonBytes()
	if err != nil {
		return nil, errors.Wrap(err, "payload bytes")
	}

	//return blocks.NewBlock(data), nil
	return blocks.NewBlockWithCid(data, cid.NewCidV1(0x55, ipfsutils.Hash(data)))
}

func (u *UploadProbe) trackProvider(ctx context.Context, provider peer.AddrInfo) error {
	if provider.ID == "" {
		u.logEntry().Warnln("Provider ID is empty")
		return nil
	}
	// TODO: go routine?
	u.trackCount += 1
	stats.Record(ctx, metrics.TrackCount.M(u.trackCount))

	err := u.host.Connect(ctx, provider)
	if err != nil {
		return errors.Wrap(err, "connect to provider")
	}

	ps := u.host.Peerstore()

	protocols, err := ps.GetProtocols(provider.ID)
	if err != nil {
		protocols = nil
	}

	var agentVersion string
	if val, err := ps.Get(provider.ID, "AgentVersion"); err == nil {
		agentVersion = val.(string)
	}

	maddrSet := map[string]ma.Multiaddr{}
	for _, maddr := range ps.Addrs(provider.ID) {
		maddrSet[maddr.String()] = maddr
	}
	for _, conn := range u.host.Network().ConnsToPeer(provider.ID) {
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
		maddrInfos, err := u.mmc.MaddrInfo(ctx, maddr)
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

	return insertModel(ctx, u.config.Database.DryRun, u.dbc, u.logEntry(), protocols, agentVersion, provider.ID,
		ipAddresses, maddrStrs, countries, continents, asns, u.target.Type(), u.target.Name())

}

func (u *UploadProbe) logEntry() *log.Entry {
	return log.WithField("type", u.target.Type()).WithField("name", u.target.Name())
}

func (u *UploadProbe) notify(err error, dur time.Duration) {
	u.logEntry().WithError(err).WithField("dur", dur).Debugln("Probe operation failed")
}

func (u *UploadProbe) wait() {
	<-u.done
}
