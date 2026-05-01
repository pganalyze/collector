package pgxdriver

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"net"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// pgxDriver is a database/sql driver that wraps pgx with a custom dial
// function for Cloud SQL or AlloyDB IAM authentication. It uses
// QueryExecModeSimpleProtocol so queries are sent via the simple query
// protocol, which is required when these managed services sit behind a
// transaction-mode connection pooler that cannot reuse prepared statements
// across pooled backend connections.
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
	config.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	config.DialFunc = func(ctx context.Context, _, _ string) (net.Conn, error) {
		return p.dial(ctx, instConnName)
	}

	dbURI := stdlib.RegisterConnConfig(config)
	p.dbURIs[name] = dbURI
	return dbURI, nil
}

// RegisterDriver registers a database/sql driver with the given name for
// Cloud SQL or AlloyDB IAM authentication. The provided dial function
// handles the actual connection through the managed service's connector.
func RegisterDriver(name string, dial func(ctx context.Context, inst string) (net.Conn, error)) {
	sql.Register(name, &pgxDriver{
		dial:   dial,
		dbURIs: make(map[string]string),
	})
}
