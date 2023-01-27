package start

import (
	"context"
	"database/sql"
	"github.com/cenkalti/backoff/v4"
	"github.com/dennis-tra/antares/pkg/db"
	"github.com/dennis-tra/antares/pkg/models"
	"github.com/dennis-tra/antares/pkg/utils"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"time"
)

type Probe interface {
	run(ctx context.Context)
	logEntry() *log.Entry
	wait()
}

func backoffWrap(ctx context.Context, c cid.Cid, fn func(context.Context, cid.Cid) error) backoff.Operation {
	return func() error {
		return fn(ctx, c)
	}
}

func cleanupProbe(ctx context.Context, logEntry *log.Entry, target Target, cid cid.Cid) {
	switch target.(type) {
	case CleanupTarget:
		op := backoffWrap(ctx, cid, target.(CleanupTarget).CleanUp)
		bo := target.Backoff(ctx)

		if err := backoff.Retry(op, bo); err != nil && !utils.IsContextErr(err) {
			logEntry.WithError(err).Warnln("Error cleaning up resources")
		}
		return
	}
	logEntry.Warnln("Target does not support cleanup")
}

func insertModel(ctx context.Context, dryRun bool, dbc *db.Client, logEntry *log.Entry, protocols []string,
	agentVersion string, peerID peer.ID, ipAddresses []string, maddrStrs []string,
	countries []string, continents []string, asns []int64, targetType string, targetName string) error {

	if dryRun {
		logEntry.Infoln("Skipping database interaction due to --dry-run flag")

		logEntry.Infoln("Tracked the following peer:")
		logEntry.Infoln("  PeerID", peerID.String())
		logEntry.Infoln("  AgentVersion", agentVersion)
		logEntry.Infoln("  Protocols", protocols)
		for i, protocol := range protocols {
			logEntry.Infof("    [%d] %s\n", i, protocol)
		}
		logEntry.Infoln("  MultiAddresses", maddrStrs)
		for i, maddrStr := range maddrStrs {
			logEntry.Infof("    [%d] %s\n", i, maddrStr)
		}
		logEntry.Infoln("  IPAddresses", ipAddresses)
		for i, ipAddress := range ipAddresses {
			logEntry.Infof("    [%d] %s\n", i, ipAddress)
		}
		logEntry.Infoln("  Countries", countries)
		logEntry.Infoln("  Continents", continents)
		logEntry.Infoln("  ASNs", asns)
		logEntry.Infoln("  TargetType", targetType)
		logEntry.Infoln("  TargetName", targetName)

		return nil
	}

	txn, err := dbc.BeginTx(ctx, nil)
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
		models.PeerWhere.TargetName.EQ(targetName),
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
			TargetType:     targetType,
			TargetName:     targetName,
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
