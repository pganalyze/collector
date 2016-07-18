// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package syslog provides a simple interface to the system log service.
package syslog

import (
	"crypto/tls"
	"errors"
	"net"
)

// BUG(brainman): This package is not implemented on Windows yet.

func localSyslog() (conn serverConn, err error) {
	return nil, errors.New("Local syslog not implemented on windows")
}

func dial(network, address string, tlsCfg *tls.Config) (net.Conn, error) {
	if network != "tcp" && network != "udp" {
		return nil, errors.New("Invalid protocol. Windows supports tcp and udp")
	}
	if tlsCfg != nil && network == "tcp" {
		return tls.Dial(network, address, tlsCfg)
	} else {
		return net.Dial(network, address)
	}
}
