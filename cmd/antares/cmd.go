package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/dennis-tra/antares/pkg/config"

	_ "net/http/pprof"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

var (
	// RawVersion and build tag of the
	// PCP command line tool. This is
	// replaced on build via e.g.:
	// -ldflags "-X main.RawVersion=${VERSION}"
	RawVersion  = "dev"
	ShortCommit = "5f3759df" // quake
)

func main() {
	// ShortCommit version tag
	verTag := fmt.Sprintf("v%s+%s", RawVersion, ShortCommit)

	app := &cli.App{
		Name:      "antares",
		Usage:     "A tool that can detect peer information of gateways and pinning services.",
		UsageText: "antares [global options] command [command options] [arguments...]",
		Authors: []*cli.Author{
			{
				Name:  "Dennis Trautwein",
				Email: "antares@dtrautwein.eu",
			},
		},
		Version: verTag,
		Before:  Before,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "debug",
				Usage:   "Set this flag to enable debug logging",
				EnvVars: []string{"ANTARES_DEBUG"},
			},
			&cli.IntFlag{
				Name:        "log-level",
				Usage:       "Set this flag to a value from 0 (least verbose) to 6 (most verbose). Overrides the --debug flag",
				EnvVars:     []string{"ANTARES_LOG_LEVEL"},
				DefaultText: "4",
				Value:       4,
			},
			&cli.StringFlag{
				Name:    "config",
				Usage:   "Load configuration from `FILE`",
				EnvVars: []string{"ANTARES_CONFIG_FILE"},
			},
			&cli.StringFlag{
				Name:        "host",
				Usage:       "On which network interface should Antares listen on",
				EnvVars:     []string{"ANTARES_HOST"},
				DefaultText: config.DefaultConfig.Host,
				Value:       config.DefaultConfig.Host,
			},
			&cli.IntFlag{
				Name:        "port",
				Usage:       "On which port should Antares listen on",
				EnvVars:     []string{"ANTARES_Port"},
				DefaultText: strconv.Itoa(config.DefaultConfig.Port),
				Value:       config.DefaultConfig.Port,
			},
			&cli.IntFlag{
				Name:        "prom-port",
				Usage:       "On which port should prometheus serve the metrics endpoint",
				EnvVars:     []string{"ANTARES_PROMETHEUS_PORT"},
				DefaultText: strconv.Itoa(config.DefaultConfig.Prometheus.Port),
				Value:       config.DefaultConfig.Prometheus.Port,
			},
			&cli.StringFlag{
				Name:        "prom-host",
				Usage:       "Where should prometheus serve the metrics endpoint",
				EnvVars:     []string{"ANTARES_PROMETHEUS_HOST"},
				DefaultText: config.DefaultConfig.Prometheus.Host,
				Value:       config.DefaultConfig.Prometheus.Host,
			},
			&cli.IntFlag{
				Name:        "pprof-port",
				Usage:       "Port for the pprof profiling endpoint",
				EnvVars:     []string{"ANTARES_PPROF_PORT"},
				DefaultText: "2003",
				Value:       2003,
			},
			&cli.BoolFlag{
				Name:    "dry-run",
				Usage:   "Don't persist anything to a database (you don't need a running DB)",
				EnvVars: []string{"ANTARES_DATABASE_DRY_RUN"},
			},
			&cli.StringFlag{
				Name:        "db-host",
				Usage:       "On which host address can antares reach the database",
				EnvVars:     []string{"ANTARES_DATABASE_HOST"},
				DefaultText: config.DefaultConfig.Database.Host,
				Value:       config.DefaultConfig.Database.Host,
			},
			&cli.IntFlag{
				Name:        "db-port",
				Usage:       "On which port can antares reach the database",
				EnvVars:     []string{"ANTARES_DATABASE_PORT"},
				DefaultText: strconv.Itoa(config.DefaultConfig.Database.Port),
				Value:       config.DefaultConfig.Database.Port,
			},
			&cli.StringFlag{
				Name:        "db-name",
				Usage:       "The name of the database to use",
				EnvVars:     []string{"ANTARES_DATABASE_NAME"},
				DefaultText: config.DefaultConfig.Database.Name,
				Value:       config.DefaultConfig.Database.Name,
			},
			&cli.StringFlag{
				Name:        "db-password",
				Usage:       "The password for the database to use",
				EnvVars:     []string{"ANTARES_DATABASE_PASSWORD"},
				DefaultText: config.DefaultConfig.Database.Password,
				Value:       config.DefaultConfig.Database.Password,
			},
			&cli.StringFlag{
				Name:        "db-user",
				Usage:       "The user with which to access the database to use",
				EnvVars:     []string{"ANTARES_DATABASE_USER"},
				DefaultText: config.DefaultConfig.Database.User,
				Value:       config.DefaultConfig.Database.User,
			},
			&cli.StringFlag{
				Name:        "db-sslmode",
				Usage:       "The sslmode to use when connecting the the database",
				EnvVars:     []string{"ANTARES_DATABASE_SSL_MODE"},
				DefaultText: config.DefaultConfig.Database.SSLMode,
				Value:       config.DefaultConfig.Database.SSLMode,
			},
			&cli.StringSliceFlag{
				Name:    "bootstrap-peers",
				Usage:   "Comma separated list of multi addresses of bootstrap peers",
				EnvVars: []string{"ANTARES_BOOTSTRAP_PEERS"},
			},
		},
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			StartCommand,
		},
	}

	sigs := make(chan os.Signal, 1)
	ctx, cancel := context.WithCancel(context.Background())

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	go func() {
		sig := <-sigs
		log.Printf("Received %s signal - Stopping...\n", sig.String())
		signal.Stop(sigs)
		cancel()
	}()

	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Errorf("error: %v\n", err)
		os.Exit(1)
	}
}

// Before is executed before any subcommands are run, but after the context is ready
// If a non-nil error is returned, no subcommands are run.
func Before(c *cli.Context) error {
	if c.Bool("debug") {
		log.SetLevel(log.DebugLevel)
	}

	if c.IsSet("log-level") {
		ll := c.Int("log-level")
		log.SetLevel(log.Level(ll))
		if ll == int(log.TraceLevel) {
			boil.DebugMode = true
		}
	}

	go func() {
		pprof := fmt.Sprintf("0.0.0.0:%d", c.Int("pprof-port"))
		log.Infoln("Starting profiling endpoint at", pprof)
		if err := http.ListenAndServe(pprof, nil); err != nil {
			log.WithError(err).Warnln("Error serving pprof")
		}
	}()

	return nil
}
