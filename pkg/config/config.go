package config

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/libp2p/go-libp2p/core/crypto"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

const (
	// Prefix is used to determine the XDG config directory.
	Prefix = "antares"
)

// configFile contains the path suffix that's appended to
// an XDG compliant directory to find the settings file.
var configFile = filepath.Join(Prefix, "config.json")

// DefaultConfig the default configuration.
var DefaultConfig = Config{
	Host: "0.0.0.0",
	Port: 2002,
	Prometheus: struct {
		Host string
		Port int
	}{
		Host: "0.0.0.0",
		Port: 2004,
	},
	Database: struct {
		DryRun   bool
		Host     string
		Port     int
		Name     string
		Password string
		User     string
		SSLMode  string
	}{
		DryRun:   false,
		Host:     "0.0.0.0",
		Port:     5432,
		Name:     "antares",
		Password: "password",
		User:     "antares",
		SSLMode:  "disable",
	},
	PrivKeyRaw:      nil,
	PinningServices: []PinningService{},
	Gateways:        []Gateway{},
	UploadServices:  []UploadService{},
}

// Config contains general user configuration.
type Config struct {
	// The version string of nebula
	Version string `json:"-"`

	// The path where the configuration file is located.
	Path string `json:"-"`

	// Whether the configuration file existed when nebula was started
	Existed bool `json:"-"`

	// Determines the IPv4 network interface Antares should bind to
	Host string

	// Determines the port at which Antares will be reachable.
	Port int

	// TODO
	ListenAddrTCP ma.Multiaddr `json:"-"`

	// TODO
	ListenAddrQUIC ma.Multiaddr `json:"-"`

	// Prometheus contains the prometheus configuration
	Prometheus struct {
		// Determines the prometheus network interface to bind to.
		Host string

		// Determines the port where prometheus serves the metrics endpoint.
		Port int
	}

	// Database contains the database connection configuration
	Database struct {
		// Whether a database connection should be established or not
		DryRun bool

		// Determines the host address of the database.
		Host string

		// Determines the port of the database.
		Port int

		// Determines the name of the database that should be used.
		Name string

		// Determines the password with which we access the database.
		Password string

		// Determines the username with which we access the database.
		User string

		// Postgres SSL mode (should be one supported in https://www.postgresql.org/docs/current/libpq-ssl.html)
		SSLMode string
	}

	// TODO
	PrivKeyRaw []byte

	// TODO
	PrivKey crypto.PrivKey `json:"-"`

	// TODO
	PinningServices []PinningService

	// TODO
	Gateways []Gateway

	UploadServices []UploadService
}

type PinningService struct {
	Target        string
	Authorization string
}

type Gateway struct {
	Name string
	URL  string
}

type UploadService struct {
	Target        string
	Authorization string
}

// Init takes the command line argument and tries to read the config file from that directory.
func Init(c *cli.Context) (*Config, error) {
	conf, err := read(c.String("config"))
	if err != nil {
		return nil, errors.Wrap(err, "read config")
	}

	// Apply command line argument configurations.
	conf.apply(c)

	// Print full configuration.
	log.Debugln("Configuration (CLI params overwrite file config):\n", conf)

	// Populate the context with the configuration.
	return conf, nil
}

// Save persists the configuration to disk using the `Path` field.
// Permissions will be 0o744
func (c *Config) Save() error {
	log.Infoln("Saving configuration file to", c.Path)

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	if c.Path == "" {
		c.Path, err = xdg.ConfigFile(configFile)
		if err != nil {
			return err
		}
	}

	return os.WriteFile(c.Path, data, 0o744)
}

// String prints the configuration as a json string
func (c *Config) String() string {
	data, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("%s", data)
}

func read(path string) (*Config, error) {
	if path == "" {
		// If no configuration file was given use xdg file.
		var err error
		path, err = xdg.ConfigFile(configFile)
		if err != nil {
			return nil, err
		}
	}

	log.Infoln("Loading configuration from:", path)
	conf := DefaultConfig
	conf.Path = path
	data, err := os.ReadFile(path)
	if err == nil {
		err = json.Unmarshal(data, &conf)
		if err != nil {
			return nil, errors.Wrap(err, "unmarshal configuration")
		}
		conf.Existed = true

		if len(conf.PrivKeyRaw) == 0 {
			log.Infoln("Config found but generating new peer identity...")
			conf.PrivKey, _, err = crypto.GenerateEd25519Key(rand.Reader)
			if err != nil {
				return nil, errors.Wrap(err, "generate key pair")
			}

			conf.PrivKeyRaw, err = crypto.MarshalPrivateKey(conf.PrivKey)
			if err != nil {
				return nil, errors.Wrap(err, "raw private key")
			}
		} else {
			conf.PrivKey, err = crypto.UnmarshalPrivateKey(conf.PrivKeyRaw)
			if err != nil {
				return nil, errors.Wrap(err, "unmarshal private key")
			}
		}

		conf.ListenAddrTCP, err = ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", conf.Host, conf.Port))
		if err != nil {
			return nil, errors.Wrap(err, "construct IPv4 TCP address")
		}

		conf.ListenAddrQUIC, err = ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/udp/%d/quic", conf.Host, conf.Port))
		if err != nil {
			return nil, errors.Wrap(err, "construct IPv4 QUIC address")
		}

		return &conf, conf.Save()
	} else if !os.IsNotExist(err) {
		return nil, err
	} else {

		conf.PrivKey, _, err = crypto.GenerateEd25519Key(rand.Reader)
		if err != nil {
			return nil, errors.Wrap(err, "generate key pair")
		}

		conf.PrivKeyRaw, err = crypto.MarshalPrivateKey(conf.PrivKey)
		if err != nil {
			return nil, errors.Wrap(err, "raw private key")
		}

		return &conf, conf.Save()
	}
}

// apply takes command line arguments and overwrites the respective configurations.
func (c *Config) apply(ctx *cli.Context) {
	c.Version = ctx.App.Version

	if ctx.IsSet("host") {
		c.Host = ctx.String("host")
	}
	if ctx.IsSet("port") {
		c.Port = ctx.Int("port")
	}
	if ctx.IsSet("prom-host") {
		c.Prometheus.Host = ctx.String("prom-host")
	}
	if ctx.IsSet("prom-port") {
		c.Prometheus.Port = ctx.Int("prom-port")
	}
	if ctx.IsSet("dry-run") {
		c.Database.DryRun = ctx.Bool("dry-run")
	}
	if ctx.IsSet("db-host") {
		c.Database.Host = ctx.String("db-host")
	}
	if ctx.IsSet("db-port") {
		c.Database.Port = ctx.Int("db-port")
	}
	if ctx.IsSet("db-name") {
		c.Database.Name = ctx.String("db-name")
	}
	if ctx.IsSet("db-password") {
		c.Database.Password = ctx.String("db-password")
	}
	if ctx.IsSet("db-user") {
		c.Database.User = ctx.String("db-user")
	}
	if ctx.IsSet("db-sslmode") {
		c.Database.SSLMode = ctx.String("db-sslmode")
	}
}
