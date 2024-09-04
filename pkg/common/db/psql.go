package db

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	"time"
)

type PSQL struct {
	timeout time.Duration
	url     string
	pool    *pgxpool.Pool
}

type Rent struct {
	CarNum string
	Start  time.Time
	Stop   time.Time
}

func New(url string, timeout time.Duration) *PSQL {
	return &PSQL{timeout: timeout, url: url}
}

func (p *PSQL) Start(ctx context.Context, errCh chan error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	pool, err := pgxpool.Connect(ctxTimeout, p.url)
	if err != nil {
		errCh <- fmt.Errorf("Create PSQL connect", err)
	}

	p.pool = pool
}

func (p *PSQL) Stop() {
	p.pool.Close()
}

func (p *PSQL) CarIsFree(ctx context.Context, rent Rent, serviceDays int) (bool, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	sql := fmt.Sprintf("select count(*) from rent where gos_num = '%s' and ('%s' between \"start\"- %d and stop + %d or '%s' between \"start\" - %d and stop + %d);",
		rent.CarNum,
		rent.Start.Format(time.DateOnly),
		serviceDays, serviceDays,
		rent.Stop.Format(time.DateOnly),
		serviceDays, serviceDays)

	row := p.pool.QueryRow(ctxTimeout, sql)

	var count int
	err := row.Scan(&count)
	if err != nil {
		return false, err
	}
	if count == 0 {
		return true, nil
	} else {
		return false, nil
	}
}

func (p *PSQL) NewRent(ctx context.Context, rent Rent, cost float64) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	_, err := p.pool.Exec(ctxTimeout, "insert into rent (gos_num, start, stop,cost) values($1,$2,$3,$4)", rent.CarNum, rent.Start, rent.Stop, cost)

	return err
}
