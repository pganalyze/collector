// Copyright 2023 Google LLC
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

// Package alloydbconn provides functions for authorizing and encrypting
// connections. These functions can be used with a database driver to
// connect to an AlloyDB cluster.
//
// # Creating a Dialer
//
// To start working with this package, create a Dialer. There are two ways of
// creating a Dialer, which one you use depends on your database driver.
//
// Users have the option of using the [database/sql] interface or using [pgx] directly.
//
// To use a dialer with [pgx], we recommend using connection pooling with
// [pgxpool]. To create the dialer use the NewDialer func.
//
//	import (
//	    "context"
//	    "net"
//
//	    "cloud.google.com/go/alloydbconn"
//	    "github.com/jackc/pgx/v4/pgxpool"
//	)
//
//	func connect() {
//	    // Configure the driver to connect to the database
//	    dsn := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", pgUser, pgPass, pgDB)
//	    config, err := pgxpool.ParseConfig(dsn)
//	    if err != nil {
//	        log.Fatalf("failed to parse pgx config: %v", err)
//	    }
//
//	    // Create a new dialer with any options
//	    d, err := alloydbconn.NewDialer(ctx)
//	    if err != nil {
//	        log.Fatalf("failed to initialize dialer: %v", err)
//	    }
//	    defer d.Close()
//
//	    // Tell the driver to use the AlloyDB Go Connector to create connections
//	    config.ConnConfig.DialFunc = func(ctx context.Context, _ string, instance string) (net.Conn, error) {
//	        return d.Dial(ctx, "projects/<PROJECT>/locations/<REGION>/clusters/<CLUSTER>/instances/<INSTANCE>")
//	    }
//
//	    // Interact with the driver directly as you normally would
//	    conn, err := pgxpool.ConnectConfig(context.Background(), config)
//	    if err != nil {
//	        log.Fatalf("failed to connect: %v", connErr)
//	    }
//	    defer conn.Close()
//	}
//
// To use [database/sql], call pgxv4.RegisterDriver with any necessary Dialer
// configuration.
//
//	import (
//	    "database/sql"
//
//	    "cloud.google.com/go/alloydbconn"
//	    "cloud.google.com/go/alloydbconn/driver/pgxv4"
//	)
//
//	func connect() {
//	    // adjust options as needed
//	    cleanup, err := pgxv4.RegisterDriver("alloydb")
//	    if err != nil {
//	        // ... handle error
//	    }
//	    defer cleanup()
//
//	    db, err := sql.Open(
//	        "alloydb",
//	        "host=projects/<PROJECT>/locations/<REGION>/clusters/<CLUSTER>/instances/<INSTANCE> user=myuser password=mypass dbname=mydb sslmode=disable",
//	    )
//	    //... etc
//	}
//
// [database/sql]: https://pkg.go.dev/database/sql
// [pgx]: https://github.com/jackc/pgx
// [pgxpool]: https://pkg.go.dev/github.com/jackc/pgx/v4/pgxpool
package alloydbconn
