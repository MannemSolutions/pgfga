package ldap

import (
	"go.uber.org/zap"
)

var log *zap.SugaredLogger

func Initialize(sugar *zap.SugaredLogger) {
	log = sugar
}
