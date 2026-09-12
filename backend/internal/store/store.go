package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	Pool *pgxpool.Pool
}

type Request struct {
	RequestID  string
	TS         time.Time
	Service    string
	Endpoint   string
	Method     string
	StatusCode int
	LatencyMS  int
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return &Store{Pool: pool}, nil
}

func (s *Store) Count(ctx context.Context) (int64, error) {
	var n int64
	err := s.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM api_requests`).Scan(&n)
	return n, err
}

func (s *Store) Close() {
	if s != nil && s.Pool != nil {
		s.Pool.Close()
	}
}

const insertSQL = `
INSERT INTO api_requests (request_id, ts, service, endpoint, method, status_code, latency_ms)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (request_id) DO NOTHING
`

const insertBatchSize = 500

func (s *Store) InsertRequests(ctx context.Context, rows []Request) (inserted int, dbDuplicates []string, err error) {
	for start := 0; start < len(rows); start += insertBatchSize {
		end := start + insertBatchSize
		if end > len(rows) {
			end = len(rows)
		}
		batch := rows[start:end]
		pgBatch := &pgx.Batch{}
		for _, row := range batch {
			pgBatch.Queue(insertSQL, row.RequestID, row.TS, row.Service, row.Endpoint, row.Method, row.StatusCode, row.LatencyMS)
		}
		results := s.Pool.SendBatch(ctx, pgBatch)
		for _, row := range batch {
			tag, execErr := results.Exec()
			if execErr != nil {
				_ = results.Close()
				return inserted, dbDuplicates, execErr
			}
			if tag.RowsAffected() == 0 {
				dbDuplicates = append(dbDuplicates, row.RequestID)
				continue
			}
			inserted++
		}
		if err := results.Close(); err != nil {
			return inserted, dbDuplicates, err
		}
	}
	return inserted, dbDuplicates, nil
}
