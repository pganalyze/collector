package state

import "errors"

var ErrReplicaCollectionDisabled error = errors.New("monitored server is replica and replica collection disabled via config")
