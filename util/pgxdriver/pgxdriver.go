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
	config.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	config.DialFunc = func(ctx context.Context, _, _ string) (net.Conn, error) {
		return p.dial(ctx, instConnName)
	}

	dbURI := stdlib.RegisterConnConfig(config)
	p.dbURIs[name] = dbURI
	return dbURI, nil
}

// RegisterDriver registers a database/sql driver with the given name that
// uses pgx with QueryExecModeExec to avoid creating server-side prepared
// statements. The provided dial function handles the actual connection.
func RegisterDriver(name string, dial func(ctx context.Context, inst string) (net.Conn, error)) {
	sql.Register(name, &pgxDriver{
		dial:   dial,
		dbURIs: make(map[string]string),
	})
}
