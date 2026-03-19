package pgxdriver

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"net"
	"sync"

	"cloud.google.com/go/alloydbconn"
	"cloud.google.com/go/cloudsqlconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// pgxDriver is a database/sql driver that uses pgx with QueryExecModeExec
// to avoid creating server-side prepared statements. It wraps a dial function
// (Cloud SQL or AlloyDB) that handles the actual connection.
type pgxDriver struct {
	dial   func(ctx context.Context, inst string) (net.Conn, error)
	mu     sync.Mutex
	dbURIs map[string]string
}

func (p *pgxDriver) Open(name string) (driver.Conn, error) {
	dbURI, err := p.dbURI(name)
	if err != nil {
		return nil, err
	}
	return stdlib.GetDefaultDriver().Open(dbURI)
}

func (p *pgxDriver) dbURI(name string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if dbURI, ok := p.dbURIs[name]; ok {
		return dbURI, nil
	}

	config, err := pgx.ParseConfig(name)
	if err != nil {
		return "", err
	}
	instConnName := config.Config.Host
	config.Config.Host = "localhost"
	config.DefaultQueryExecMode = pgx.QueryExecModeExec
	config.DialFunc = func(ctx context.Context, _, _ string) (net.Conn, error) {
		return p.dial(ctx, instConnName)
	}

	dbURI := stdlib.RegisterConnConfig(config)
	p.dbURIs[name] = dbURI
	return dbURI, nil
}

// RegisterCloudSQLDriver registers a database/sql driver for Cloud SQL that
// uses pgx with QueryExecModeExec to avoid creating server-side prepared
// statements.
func RegisterCloudSQLDriver(name string, opts ...cloudsqlconn.Option) (func() error, error) {
	d, err := cloudsqlconn.NewDialer(context.Background(), opts...)
	if err != nil {
		return func() error { return nil }, err
	}
	sql.Register(name, &pgxDriver{
		dial:   func(ctx context.Context, inst string) (net.Conn, error) { return d.Dial(ctx, inst) },
		dbURIs: make(map[string]string),
	})
	return func() error { return d.Close() }, nil
}

// RegisterAlloyDBDriver registers a database/sql driver for AlloyDB that
// uses pgx with QueryExecModeExec to avoid creating server-side prepared
// statements.
func RegisterAlloyDBDriver(name string, opts ...alloydbconn.Option) (func() error, error) {
	d, err := alloydbconn.NewDialer(context.Background(), opts...)
	if err != nil {
		return func() error { return nil }, err
	}
	sql.Register(name, &pgxDriver{
		dial:   func(ctx context.Context, inst string) (net.Conn, error) { return d.Dial(ctx, inst) },
		dbURIs: make(map[string]string),
	})
	return func() error { return d.Close() }, nil
}
