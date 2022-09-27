package db

import (
	"context"
	"database/sql"
	"fmt"

	"contrib.go.opencensus.io/integrations/ocsql"

	"github.com/dennis-tra/antares/pkg/config"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type Client struct {
	// Database handler
	dbh *sql.DB
}

func InitClient(conf *config.Config) (*Client, error) {
	log.WithFields(log.Fields{
		"host": conf.Database.Host,
		"port": conf.Database.Port,
		"name": conf.Database.Name,
		"user": conf.Database.User,
		"ssl":  conf.Database.SSLMode,
	}).Infoln("Initializing database client")

	driverName, err := ocsql.Register("postgres")
	if err != nil {
		return nil, errors.Wrap(err, "register ocsql")
	}

	// Open database handle
	srcName := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		conf.Database.Host,
		conf.Database.Port,
		conf.Database.Name,
		conf.Database.User,
		conf.Database.Password,
		conf.Database.SSLMode,
	)
	dbh, err := sql.Open(driverName, srcName)
	if err != nil {
		return nil, errors.Wrap(err, "opening database")
	}

	// Ping database to verify connection.
	if err = dbh.Ping(); err != nil {
		return nil, errors.Wrap(err, "pinging database")
	}

	return &Client{dbh}, nil
}

func (c *Client) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return c.dbh.BeginTx(ctx, opts)
}
