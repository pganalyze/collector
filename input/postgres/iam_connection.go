package postgres

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"cloud.google.com/go/alloydbconn"
	"cloud.google.com/go/cloudsqlconn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds/rdsutils"
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/util/awsutil"
	"github.com/pganalyze/collector/util/pgxdriver"
)

// Overrides that IAM auth needs to apply to a connection
type iamConnectionParams struct {
	hostOverride     string
	passwordOverride string
	sslmodeOverride  string
}

func getIamConnectionParams(config config.ServerConfig) (driverName string, iamParams iamConnectionParams, err error) {
	switch config.SystemType {
	case "amazon_rds":
		var sess *session.Session
		sess, err = awsutil.GetAwsSession(config)
		if err != nil {
			return
		}
		var dbToken string
		dbToken, err = rdsutils.BuildAuthToken(
			fmt.Sprintf("%s:%d", config.GetDbHost(), config.GetDbPortOrDefault()),
			config.AwsRegion,
			config.GetDbUsername(),
			sess.Config.Credentials,
		)
		if err != nil {
			return
		}

		driverName = "postgres"
		iamParams.passwordOverride = dbToken

	case "google_cloudsql":
		if config.GcpProjectID == "" || config.GcpRegion == "" {
			err = errors.New("To use IAM auth with Google Cloud SQL or AlloyDB, you must specify project ID and region in the configuration")
			return
		}
		if config.GcpCloudSQLInstanceID != "" {
			iamParams.hostOverride = strings.Join([]string{config.GcpProjectID, config.GcpRegion, config.GcpCloudSQLInstanceID}, ":")
			if config.GcpUsePSC {
				driverName = "cloudsql-postgres-psc"
			} else if config.GcpUsePublicIP {
				driverName = "cloudsql-postgres-public"
			} else {
				driverName = "cloudsql-postgres"
			}
		} else if config.GcpAlloyDBClusterID != "" && config.GcpAlloyDBInstanceID != "" {
			iamParams.hostOverride = fmt.Sprintf("projects/%s/locations/%s/clusters/%s/instances/%s", config.GcpProjectID, config.GcpRegion, config.GcpAlloyDBClusterID, config.GcpAlloyDBInstanceID)
			if config.GcpUsePSC {
				driverName = "alloydb-postgres-psc"
			} else if config.GcpUsePublicIP {
				driverName = "alloydb-postgres-public"
			} else {
				driverName = "alloydb-postgres"
			}
		} else {
			err = errors.New("To use IAM auth with either Google Cloud SQL or AlloyDB, you must specify instance ID (CloudSQL) or cluster ID and instance ID (AlloyDB) in the configuration")
			return
		}

		// IAM connections go through cloud-sql-go-connector which does its own
		// mTLS handling, so sslmode needs to be set as "disable"
		//
		// See https://github.com/GoogleCloudPlatform/cloud-sql-go-connector/issues/889
		iamParams.sslmodeOverride = "disable"

	default:
		err = errors.New("IAM auth is only supported for Amazon RDS, Aurora, Google Cloud SQL, and Google AlloyDB - turn off IAM auth setting to use password-based authentication")
		return
	}

	return
}

// RegisterCloudSQLDrivers registers database/sql drivers needed for
// IAM-authenticated connections to Cloud SQL.
func RegisterCloudSQLDrivers() (cleanups []func() error, err error) {
	register := func(name string, opts ...cloudsqlconn.Option) error {
		d, err := cloudsqlconn.NewDialer(context.Background(), opts...)
		if err != nil {
			return err
		}
		pgxdriver.RegisterDriver(name, func(ctx context.Context, inst string) (net.Conn, error) { return d.Dial(ctx, inst) })
		cleanups = append(cleanups, d.Close)
		return nil
	}

	err = register("cloudsql-postgres", cloudsqlconn.WithIAMAuthN(), cloudsqlconn.WithDefaultDialOptions(cloudsqlconn.WithPrivateIP()))
	if err != nil {
		return
	}
	err = register("cloudsql-postgres-public", cloudsqlconn.WithIAMAuthN()) // CloudSQL defaults to public IP
	if err != nil {
		return
	}
	err = register("cloudsql-postgres-psc", cloudsqlconn.WithIAMAuthN(), cloudsqlconn.WithDefaultDialOptions(cloudsqlconn.WithPSC()))
	if err != nil {
		return
	}

	return
}

// RegisterAlloyDBDrivers registers database/sql drivers needed for
// IAM-authenticated connections to AlloyDB.
func RegisterAlloyDBDrivers() (cleanups []func() error, err error) {
	register := func(name string, opts ...alloydbconn.Option) error {
		d, err := alloydbconn.NewDialer(context.Background(), opts...)
		if err != nil {
			return err
		}
		pgxdriver.RegisterDriver(name, func(ctx context.Context, inst string) (net.Conn, error) { return d.Dial(ctx, inst) })
		cleanups = append(cleanups, d.Close)
		return nil
	}

	err = register("alloydb-postgres", alloydbconn.WithIAMAuthN()) // AlloyDB defaults to private IP
	if err != nil {
		return
	}
	err = register("alloydb-postgres-public", alloydbconn.WithIAMAuthN(), alloydbconn.WithDefaultDialOptions(alloydbconn.WithPublicIP()))
	if err != nil {
		return
	}
	err = register("alloydb-postgres-psc", alloydbconn.WithIAMAuthN(), alloydbconn.WithDefaultDialOptions(alloydbconn.WithPSC()))
	if err != nil {
		return
	}

	return
}
