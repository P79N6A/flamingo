package gorm

import (
	"context"
	"database/sql"
)

// Context set context
func (s *DB) Context(ctx context.Context) *DB {
	clone := s.clone()
	clone.Ctx = ctx
	return clone
}

type sqlCtxExecer interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

type sqlCtxQuerier interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

type sqlCtxPreparer interface {
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}
