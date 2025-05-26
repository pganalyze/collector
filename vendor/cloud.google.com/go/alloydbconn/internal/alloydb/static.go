// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package alloydb

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"io"

	"cloud.google.com/go/alloydbconn/debug"
	"cloud.google.com/go/alloydbconn/errtype"
)

type staticPSCConfig struct {
	PSCDNSName string `json:"pscDnsName"`
}

// staticConnectionInfo is an amalgamation of the generate ephemeral
// certificate and instance metadata endpoints. Its structure concatenates the
// IP address information with certificate information. As such it provides all
// the necessary properties needed for the Dialer to connect to an instance's
// Auth Proxy server.
type staticConnectionInfo struct {
	IPAddress           string          `json:"ipAddress"`
	PublicIPAddress     string          `json:"publicIPAddress"`
	PSCInstanceConfig   staticPSCConfig `json:"pscInstanceConfig"`
	PEMCertificateChain []string        `json:"pemCertificateChain"`
	CACert              string          `json:"caCert"`
}

// staticInstanceInfo correlates instance URIs with static connection info.
type staticInstanceInfo map[string]staticConnectionInfo

// staticData represent a collection of static connection info.
type staticData struct {
	PublicKey    string
	PrivateKey   string
	InstanceInfo staticInstanceInfo
}

func (s *staticData) UnmarshalJSON(data []byte) error {
	inner := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &inner); err != nil {
		return err
	}
	if err := json.Unmarshal(inner["privateKey"], &s.PrivateKey); err != nil {
		return err
	}
	delete(inner, "privateKey")
	if err := json.Unmarshal(inner["publicKey"], &s.PublicKey); err != nil {
		return err
	}
	delete(inner, "publicKey")

	s.InstanceInfo = staticInstanceInfo{}
	for k, v := range inner {
		var sci staticConnectionInfo
		if err := json.Unmarshal(v, &sci); err != nil {
			return err
		}
		s.InstanceInfo[k] = sci
	}
	return nil
}

// StaticConnectionInfoCache provides connection info that is never refreshed.
type StaticConnectionInfoCache struct {
	logger debug.ContextLogger
	info   ConnectionInfo
}

// NewStaticConnectionInfoCache creates a connection info cache that will
// always return the predefined connection info within the provided io.Reader
func NewStaticConnectionInfoCache(
	inst InstanceURI,
	l debug.ContextLogger,
	r io.Reader,
) (*StaticConnectionInfoCache, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var d staticData
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}
	static, ok := d.InstanceInfo[inst.URI()]
	if !ok {
		return nil, errtype.NewConfigError("unknown instance", inst.String())
	}
	cc, err := newClientCertificate(
		inst, []byte(d.PrivateKey), static.PEMCertificateChain, static.CACert,
	)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	pool.AddCert(cc.caCert)
	info := ConnectionInfo{
		Instance: inst,
		IPAddrs: map[string]string{
			PublicIP:  static.PublicIPAddress,
			PrivateIP: static.IPAddress,
			PSC:       static.PSCInstanceConfig.PSCDNSName,
		},
		ClientCert: cc.certChain,
		RootCAs:    pool,
		Expiration: cc.expiry,
	}
	return &StaticConnectionInfoCache{
		logger: l,
		info:   info,
	}, nil
}

// ConnectionInfo returns the connection info for the specified instance URI as
// loaded from the provided io.Reader.
func (c *StaticConnectionInfoCache) ConnectionInfo(
	_ context.Context,
) (ConnectionInfo, error) {
	return c.info, nil
}

// ForceRefresh is a no-op as the cache holds only static connection
// information and does no refresh.
func (*StaticConnectionInfoCache) ForceRefresh() {}

// Close is a no-op.
func (*StaticConnectionInfoCache) Close() error { return nil }
