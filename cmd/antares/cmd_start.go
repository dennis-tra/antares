package main

import (
	"github.com/dennis-tra/antares/pkg/config"
	"github.com/dennis-tra/antares/pkg/db"
	"github.com/dennis-tra/antares/pkg/maxmind"
	"github.com/dennis-tra/antares/pkg/start"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// StartCommand contains the start sub-command configuration.
var StartCommand = &cli.Command{
	Name:   "start",
	Usage:  "Starts to provide content to the network and request it through gateways and pinning services.",
	Action: StartAction,
}

// StartAction is the command line action to start the libp2p host that provides content to the network
// via the DHT and then requests that content through gateways and pinning services.
func StartAction(c *cli.Context) error {
	log.Infoln("Starting Antares...")

	// Load configuration file
	conf, err := config.Init(c)
	if err != nil {
		return errors.Wrap(err, "init configuration")
	}

	// Acquire database handle
	var dbc *db.Client
	if !conf.Database.DryRun {
		if dbc, err = db.InitClient(conf); err != nil {
			return err
		}
	}

	// Initialize new maxmind client to interact with the country database.
	mmc, err := maxmind.NewClient()
	if err != nil {
		return err
	}

	// Initialize scheduler that handles crawling the network.
	s, err := start.NewScheduler(c.Context, conf, dbc, mmc)
	if err != nil {
		return errors.Wrap(err, "creating new scheduler")
	}

	return s.StartProbes(c.Context)
}
