package utils

import (
	"context"
	"net/http"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func IsRelayedMaddr(maddr ma.Multiaddr) bool {
	_, err := maddr.ValueForProtocol(ma.P_CIRCUIT)
	if err == nil {
		return true
	} else if errors.Is(err, ma.ErrProtocolNotFound) {
		return false
	} else {
		log.WithError(err).WithField("maddr", maddr).Warnln("Unexpected error while parsing multi address")
		return false
	}
}

func IsContextErr(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func IsSuccessStatusCode(res *http.Response) bool {
	return res.StatusCode >= 200 && res.StatusCode < 300
}
