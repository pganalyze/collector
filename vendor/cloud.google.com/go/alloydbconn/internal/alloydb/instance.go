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
	"context"
	"crypto/rsa"
	"fmt"
	"regexp"
	"sync"
	"time"

	alloydbadmin "cloud.google.com/go/alloydb/apiv1alpha"
	"cloud.google.com/go/alloydbconn/debug"
	"cloud.google.com/go/alloydbconn/errtype"
	telv2 "cloud.google.com/go/alloydbconn/internal/tel/v2"
	"golang.org/x/time/rate"
)

const (
	// the refresh buffer is the amount of time before a refresh cycle's result
	// expires that a new refresh operation begins.
	refreshBuffer = 4 * time.Minute

	// refreshInterval is the amount of time between refresh attempts as
	// enforced by the rate limiter.
	refreshInterval = 30 * time.Second

	// RefreshTimeout is the maximum amount of time to wait for a refresh
	// cycle to complete. This value should be greater than the
	// refreshInterval.
	RefreshTimeout = 60 * time.Second

	// refreshBurst is the initial burst allowed by the rate limiter.
	refreshBurst = 2
)

var (
	// Instance URI is in the format:
	// 'projects/<PROJECT>/locations/<REGION>/clusters/<CLUSTER>/instances/<INSTANCE>'
	// Additionally, we have to support legacy "domain-scoped" projects
	// (e.g. "google.com:PROJECT")
	instURIRegex = regexp.MustCompile("projects/([^:]+(:[^:]+)?)/locations/([^:]+)/clusters/([^:]+)/instances/([^:]+)")
)

// InstanceURI represents an AlloyDB instance.
type InstanceURI struct {
	project string
	region  string
	cluster string
	name    string
}

// Project returns the project ID of the cluster.
func (i InstanceURI) Project() string {
	return i.project
}

// Region returns the region (aka location) of the cluster.
func (i InstanceURI) Region() string {
	return i.region
}

// Cluster returns the name of the cluster.
func (i InstanceURI) Cluster() string {
	return i.cluster
}

// Name returns the name of the instance.
func (i InstanceURI) Name() string {
	return i.name
}

// URI returns the full URI specifying an instance.
func (i *InstanceURI) URI() string {
	return fmt.Sprintf(
		"projects/%s/locations/%s/clusters/%s/instances/%s",
		i.project, i.region, i.cluster, i.name,
	)
}

// String returns a short-hand representation of an instance URI.
func (i *InstanceURI) String() string {
	return fmt.Sprintf("%s/%s/%s/%s", i.project, i.region, i.cluster, i.name)
}

// ParseInstURI initializes a new InstanceURI struct.
func ParseInstURI(cn string) (InstanceURI, error) {
	b := []byte(cn)
	m := instURIRegex.FindSubmatch(b)
	if m == nil {
		err := errtype.NewConfigError(
			"invalid instance URI, expected projects/<PROJECT>/locations/<REGION>/clusters/<CLUSTER>/instances/<INSTANCE>",
			cn,
		)
		return InstanceURI{}, err
	}

	c := InstanceURI{
		project: string(m[1]),
		region:  string(m[3]),
		cluster: string(m[4]),
		name:    string(m[5]),
	}
	return c, nil
}

// refreshOperation is a pending result of a refresh operation of data used to
// connect securely. It should only be initialized by the Instance struct as
// part of a refresh cycle.
type refreshOperation struct {
	result ConnectionInfo
	err    error

	// timer that triggers refresh, can be used to cancel.
	timer *time.Timer
	// indicates the struct is ready to read from
	ready chan struct{}
}

// Cancel prevents the instanceInfo from starting, if it hasn't already
// started. Returns true if timer was stopped successfully, or false if it has
// already started.
func (r *refreshOperation) cancel() bool {
	return r.timer.Stop()
}

// IsValid returns true if this result is complete, successful, and is still
// valid.
func (r *refreshOperation) isValid() bool {
	// verify the result has finished running
	select {
	default:
		return false
	case <-r.ready:
		if r.err != nil || time.Now().After(r.result.Expiration) {
			return false
		}
		return true
	}
}

// RefreshAheadCache manages the information used to connect to the AlloyDB instance by
// periodically calling the AlloyDB Admin API. It automatically refreshes the
// required information approximately 4 minutes before the previous certificate
// expires (every ~56 minutes).
type RefreshAheadCache struct {
	instanceURI InstanceURI
	logger      debug.ContextLogger
	// refreshTimeout sets the maximum duration a refresh cycle can run
	// for.
	refreshTimeout time.Duration
	// l controls the rate at which refresh cycles are run.
	l *rate.Limiter
	r adminAPIClient

	resultGuard sync.RWMutex
	// cur represents the current refreshOperation that will be used to
	// create connections. If a valid complete refreshOperation isn't
	// available it's possible for cur to be equal to next.
	cur *refreshOperation
	// next represents a future or ongoing refreshOperation. Once complete,
	// it will replace cur and schedule a replacement to occur.
	next *refreshOperation

	// ctx is the default ctx for refresh operations. Canceling it prevents
	// new refresh operations from being triggered.
	ctx    context.Context
	cancel context.CancelFunc

	userAgent      string
	metricRecorder telv2.MetricRecorder
}

// NewRefreshAheadCache initializes a new cache that proactively refreshes the
// caches connection info.
func NewRefreshAheadCache(
	instance InstanceURI,
	l debug.ContextLogger,
	client *alloydbadmin.AlloyDBAdminClient,
	key *rsa.PrivateKey,
	refreshTimeout time.Duration,
	dialerID string,
	disableMetadataExchange bool,
	userAgent string,
	mr telv2.MetricRecorder,
) *RefreshAheadCache {
	ctx, cancel := context.WithCancel(context.Background())
	i := &RefreshAheadCache{
		instanceURI:    instance,
		logger:         l,
		l:              rate.NewLimiter(rate.Every(refreshInterval), refreshBurst),
		r:              newAdminAPIClient(client, key, dialerID, disableMetadataExchange),
		refreshTimeout: refreshTimeout,
		ctx:            ctx,
		cancel:         cancel,
		userAgent:      userAgent,
		metricRecorder: mr,
	}
	// For the initial refresh operation, set cur = next so that connection
	// requests block until the first refresh is complete.
	i.resultGuard.Lock()
	i.cur = i.scheduleRefresh(0)
	i.next = i.cur
	i.resultGuard.Unlock()
	return i
}

// Close closes the instance; it stops the refresh cycle and prevents it from
// making additional calls to the AlloyDB Admin API.
func (i *RefreshAheadCache) Close() error {
	i.resultGuard.Lock()
	defer i.resultGuard.Unlock()
	i.cancel()
	i.cur.cancel()
	i.next.cancel()
	return nil
}

// ConnectionInfo returns an IP address specified by ipType (i.e., public or
// private) of the AlloyDB instance.
func (i *RefreshAheadCache) ConnectionInfo(ctx context.Context) (ConnectionInfo, error) {
	i.resultGuard.RLock()
	refresh := i.cur
	i.resultGuard.RUnlock()
	var err error
	select {
	case <-refresh.ready:
		err = refresh.err
	case <-ctx.Done():
		err = ctx.Err()
	case <-i.ctx.Done():
		err = i.ctx.Err()
	}
	if err != nil {
		return ConnectionInfo{}, err
	}
	return refresh.result, nil
}

// ForceRefresh triggers an immediate refresh operation to be scheduled and
// used for future connection attempts if valid.
func (i *RefreshAheadCache) ForceRefresh() {
	i.resultGuard.Lock()
	defer i.resultGuard.Unlock()
	// If the next refresh hasn't started yet, we can cancel it and start an immediate one
	if i.next.cancel() {
		i.next = i.scheduleRefresh(0)
	}
	// block all sequential connection attempts on the next refresh operation
	// if current is invalid
	if !i.cur.isValid() {
		i.cur = i.next
	}
}

// refreshDuration returns the duration to wait before starting the next
// refresh. Usually that duration will be half of the time until certificate
// expiration.
func refreshDuration(now, certExpiry time.Time) time.Duration {
	d := certExpiry.Sub(now)
	if d < time.Hour {
		// Something is wrong with the certification, refresh now.
		if d < refreshBuffer {
			return 0
		}
		// Otherwise wait until 4 minutes before expiration for next refresh cycle.
		return d - refreshBuffer
	}
	return d / 2
}

// scheduleRefresh schedules a refresh operation to be triggered after a given
// duration. The returned refreshOperation can be used to either Cancel or Wait
// for the operation's result.
func (i *RefreshAheadCache) scheduleRefresh(d time.Duration) *refreshOperation {
	r := &refreshOperation{}
	r.ready = make(chan struct{})
	r.timer = time.AfterFunc(d, func() {
		// instance has been closed, don't schedule anything
		if err := i.ctx.Err(); err != nil {
			i.logger.Debugf(
				context.Background(),
				"[%v] Instance is closed, stopping refresh operations",
				i.instanceURI.String(),
			)
			r.err = err
			close(r.ready)
			return
		}
		i.logger.Debugf(
			context.Background(),
			"[%v] Connection info refresh operation started",
			i.instanceURI.String(),
		)

		ctx, cancel := context.WithTimeout(i.ctx, i.refreshTimeout)
		defer cancel()

		err := i.l.Wait(ctx)
		if err != nil {
			r.err = errtype.NewDialError(
				"context was canceled or expired before refresh completed",
				i.instanceURI.String(),
				nil,
			)
			i.logger.Debugf(
				ctx,
				"[%v] Connection info refresh operation failed, err = %v",
				i.instanceURI.String(),
				r.err,
			)
		} else {
			r.result, r.err = i.r.connectionInfo(i.ctx, i.instanceURI)
			i.logger.Debugf(
				ctx,
				"[%v] Connection info refresh operation complete",
				i.instanceURI.String(),
			)
			i.logger.Debugf(
				ctx,
				"[%v] Current certificate expiration = %v",
				i.instanceURI.String(),
				r.result.Expiration.UTC().Format(time.RFC3339),
			)
		}

		close(r.ready)

		// Once the refresh is complete, update "current" with working
		// result and schedule a new refresh
		i.resultGuard.Lock()
		defer i.resultGuard.Unlock()

		// if failed, scheduled the next refresh immediately
		if r.err != nil {
			i.logger.Debugf(
				ctx,
				"[%v] Connection info refresh operation scheduled immediately",
				i.instanceURI.String(),
			)
			i.next = i.scheduleRefresh(0)
			// If the latest result is bad, avoid replacing the
			// used result while it's still valid and potentially
			// able to provide successful connections. TODO: This
			// means that errors while the current result is still
			// valid are suppressed. We should try to surface
			// errors in a more meaningful way.
			if !i.cur.isValid() {
				i.cur = r
			}
			go i.metricRecorder.RecordRefreshCount(context.Background(), telv2.Attributes{
				UserAgent:     i.userAgent,
				RefreshType:   telv2.RefreshAheadType,
				RefreshStatus: telv2.RefreshFailure,
			})
			return
		}
		// Update the current results, and schedule the next refresh in
		// the future
		i.cur = r
		t := refreshDuration(time.Now(), i.cur.result.Expiration)
		i.logger.Debugf(
			ctx,
			"[%v] Connection info refresh operation scheduled at %v (now + %v)",
			i.instanceURI.String(),
			time.Now().Add(t).UTC().Format(time.RFC3339),
			t.Round(time.Minute),
		)
		i.next = i.scheduleRefresh(t)
		go i.metricRecorder.RecordRefreshCount(context.Background(), telv2.Attributes{
			UserAgent:     i.userAgent,
			RefreshType:   telv2.RefreshAheadType,
			RefreshStatus: telv2.RefreshSuccess,
		})
	})
	return r
}
