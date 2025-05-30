// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package alloydb

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"

	alloydbadmin "cloud.google.com/go/alloydb/apiv1alpha"
	"cloud.google.com/go/alloydb/apiv1alpha/alloydbpb"
	"cloud.google.com/go/alloydbconn/errtype"
	"cloud.google.com/go/alloydbconn/internal/tel"
	"google.golang.org/protobuf/types/known/durationpb"
)

const (
	// PublicIP is the value for public IP connections.
	PublicIP = "PUBLIC"
	// PrivateIP is the value for private IP connections.
	PrivateIP = "PRIVATE"
	// PSC designates PSC-based connections.
	PSC = "PSC"
)

type instanceInfo struct {
	// ipAddrs is the instance's IP addresses
	ipAddrs map[string]string
	// uid is the instance UID
	uid string
}

// fetchInstanceInfo uses the AlloyDB Admin APIs get method to retrieve the
// information about an AlloyDB instance that is used to create secure
// connections.
func fetchInstanceInfo(
	ctx context.Context, cl *alloydbadmin.AlloyDBAdminClient, inst InstanceURI,
) (i instanceInfo, err error) {
	var end tel.EndSpanFunc
	ctx, end = tel.StartSpan(ctx, "cloud.google.com/go/alloydbconn/internal.FetchMetadata")
	defer func() { end(err) }()
	req := &alloydbpb.GetConnectionInfoRequest{
		Parent: fmt.Sprintf(
			"projects/%s/locations/%s/clusters/%s/instances/%s",
			inst.project, inst.region, inst.cluster, inst.name,
		),
	}
	resp, err := cl.GetConnectionInfo(ctx, req)
	if err != nil {
		return instanceInfo{}, errtype.NewRefreshError(
			"failed to get instance metadata", inst.String(), err,
		)
	}

	// parse any ip addresses that might be used to connect
	ipAddrs := make(map[string]string)
	if addr := resp.GetIpAddress(); addr != "" {
		ipAddrs[PrivateIP] = addr
	}
	if addr := resp.GetPublicIpAddress(); addr != "" {
		ipAddrs[PublicIP] = addr
	}
	if addr := resp.GetPscDnsName(); addr != "" {
		ipAddrs[PSC] = addr
	}

	if len(ipAddrs) == 0 {
		return instanceInfo{}, errtype.NewConfigError(
			"cannot connect to instance - it has no supported IP addresses",
			inst.String(),
		)
	}
	return instanceInfo{ipAddrs: ipAddrs, uid: resp.InstanceUid}, nil
}

var errInvalidPEM = errors.New("certificate is not a valid PEM")

func parseCert(cert string) (*x509.Certificate, error) {
	b, _ := pem.Decode([]byte(cert))
	if b == nil {
		return nil, errInvalidPEM
	}
	return x509.ParseCertificate(b.Bytes)
}

type clientCertificate struct {
	// certChain is the client certificate chained with the intermediate
	// cert(s) and CA cert.
	certChain tls.Certificate
	// ca cert is the CA certificate of the cluster
	caCert *x509.Certificate
	// expiry is the expiration of the client certificate.
	expiry time.Time
}

// fetchClientCertificate uses the AlloyDB Admin API's
// generateClientCertificate method to create a signed TLS certificate that
// authorized to connect via the AlloyDB instance's serverside proxy. The cert
// is valid for one hour.
func fetchClientCertificate(
	ctx context.Context,
	cl *alloydbadmin.AlloyDBAdminClient,
	inst InstanceURI,
	key *rsa.PrivateKey,
	disableMetadataExchange bool,
) (cc *clientCertificate, err error) {
	var end tel.EndSpanFunc
	ctx, end = tel.StartSpan(ctx, "cloud.google.com/go/alloydbconn/internal.FetchEphemeralCert")
	defer func() { end(err) }()

	buf := &bytes.Buffer{}
	k := x509.MarshalPKCS1PublicKey(&key.PublicKey)
	err = pem.Encode(buf, &pem.Block{Type: "RSA PUBLIC KEY", Bytes: k})
	if err != nil {
		return nil, err
	}
	req := &alloydbpb.GenerateClientCertificateRequest{
		Parent: fmt.Sprintf(
			"projects/%s/locations/%s/clusters/%s", inst.project, inst.region, inst.cluster,
		),
		PublicKey:           buf.String(),
		CertDuration:        durationpb.New(time.Second * 3600),
		UseMetadataExchange: !disableMetadataExchange,
	}
	resp, err := cl.GenerateClientCertificate(ctx, req)
	if err != nil {
		return nil, errtype.NewRefreshError(
			"create ephemeral cert failed",
			inst.String(),
			err,
		)
	}

	keyPEMBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	keyPEM := pem.EncodeToMemory(keyPEMBlock)

	return newClientCertificate(
		inst, keyPEM, resp.PemCertificateChain, resp.CaCert,
	)
}

func newClientCertificate(
	inst InstanceURI,
	keyPEM []byte,
	chain []string,
	caCertRaw string,
) (cc *clientCertificate, err error) {
	certPEMBlock := []byte(strings.Join(chain, "\n"))
	cert, err := tls.X509KeyPair(certPEMBlock, keyPEM)
	if err != nil {
		return nil, errtype.NewRefreshError(
			"create ephemeral cert failed",
			inst.String(),
			err,
		)
	}

	caCertPEMBlock, _ := pem.Decode([]byte(caCertRaw))
	if caCertPEMBlock == nil {
		return nil, errtype.NewRefreshError(
			"create ephemeral cert failed",
			inst.String(),
			errors.New("no PEM data found in the ca cert"),
		)
	}
	caCert, err := x509.ParseCertificate(caCertPEMBlock.Bytes)
	if err != nil {
		return nil, errtype.NewRefreshError(
			"create ephemeral cert failed",
			inst.String(),
			err,
		)
	}

	// Extract expiry from client certificate.
	clientCertPEMBlock, _ := pem.Decode([]byte(chain[0]))
	if clientCertPEMBlock == nil {
		return nil, errtype.NewRefreshError(
			"create ephemeral cert failed",
			inst.String(),
			errors.New("no PEM data found in the client cert"),
		)
	}
	clientCert, err := x509.ParseCertificate(clientCertPEMBlock.Bytes)
	if err != nil {
		return nil, errtype.NewRefreshError(
			"create ephemeral cert failed",
			inst.String(),
			err,
		)
	}
	// Save the parsed certificate as the leaf certificate, to avoid additional
	// parsing costs as part of the TLS connection.
	cert.Leaf = clientCert

	return &clientCertificate{
		certChain: cert,
		caCert:    caCert,
		expiry:    clientCert.NotAfter,
	}, nil
}

func newAdminAPIClient(
	client *alloydbadmin.AlloyDBAdminClient,
	key *rsa.PrivateKey,
	dialerID string,
	disableMetadataExchange bool,
) adminAPIClient {
	return adminAPIClient{
		client:                  client,
		key:                     key,
		dialerID:                dialerID,
		disableMetadataExchange: disableMetadataExchange,
	}
}

// adminAPIClient manages the AlloyDB Admin API access to instance metadata and
// to ephemeral certificates.
type adminAPIClient struct {
	// client provides access to the AlloyDB Admin API
	client *alloydbadmin.AlloyDBAdminClient
	// key is used to request client certificates
	key *rsa.PrivateKey
	// dialerID is the unique ID of the associated dialer.
	dialerID string
	// disableMetadataExchange is a temporary addition to ease the migration to
	// when the metadata exchange is required.
	disableMetadataExchange bool
}

// ConnectionInfo holds all the data necessary to connect to an instance.
type ConnectionInfo struct {
	Instance   InstanceURI
	IPAddrs    map[string]string
	ClientCert tls.Certificate
	RootCAs    *x509.CertPool
	Expiration time.Time
}

func (c adminAPIClient) connectionInfo(
	ctx context.Context, i InstanceURI,
) (res ConnectionInfo, err error) {

	var refreshEnd tel.EndSpanFunc
	ctx, refreshEnd = tel.StartSpan(ctx, "cloud.google.com/go/alloydbconn/internal.RefreshConnection",
		tel.AddInstanceName(i.String()),
	)
	defer func() {
		go tel.RecordRefreshResult(
			context.Background(), i.String(), c.dialerID, err,
		)
		refreshEnd(err)
	}()

	type mdRes struct {
		info instanceInfo
		err  error
	}
	mdCh := make(chan mdRes, 1)
	go func() {
		defer close(mdCh)
		c, err := fetchInstanceInfo(ctx, c.client, i)
		mdCh <- mdRes{info: c, err: err}
	}()

	type certRes struct {
		cc  *clientCertificate
		err error
	}
	certCh := make(chan certRes, 1)
	go func() {
		defer close(certCh)
		cc, err := fetchClientCertificate(ctx, c.client, i, c.key, c.disableMetadataExchange)
		certCh <- certRes{cc: cc, err: err}
	}()

	var info instanceInfo
	select {
	case r := <-mdCh:
		if r.err != nil {
			return ConnectionInfo{}, fmt.Errorf(
				"failed to get instance IP address: %w", r.err,
			)
		}
		info = r.info
	case <-ctx.Done():
		return ConnectionInfo{}, fmt.Errorf("refresh failed: %w", ctx.Err())
	}

	var cc *clientCertificate
	select {
	case r := <-certCh:
		if r.err != nil {
			return ConnectionInfo{}, fmt.Errorf(
				"fetch ephemeral cert failed: %w", r.err,
			)
		}
		cc = r.cc
	case <-ctx.Done():
		return ConnectionInfo{}, fmt.Errorf("refresh failed: %w", ctx.Err())
	}

	caCerts := x509.NewCertPool()
	caCerts.AddCert(cc.caCert)
	ci := ConnectionInfo{
		Instance:   i,
		IPAddrs:    info.ipAddrs,
		ClientCert: cc.certChain,
		RootCAs:    caCerts,
		Expiration: cc.expiry,
	}
	return ci, nil
}
