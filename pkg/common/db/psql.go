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

	err = p.createTable(ctx)
	if err != nil {
		errCh <- err
	}
}

func (p *PSQL) Stop() {
	p.pool.Close()
}

func (p *PSQL) createTable(ctx context.Context) error {
	//ctxTimeout, cancel := context.WithTimeout(ctx, p.timeout)
	//defer cancel()

	//if _, err := p.pool.Exec(ctx, "create table if not exists orders(order_id varchar(20) unique not null,data json)"); err != nil {
	//	return fmt.Errorf("Create table", err)
	//}
	fmt.Println("СОздание таблицы")
	return nil
}

func (p *PSQL) CarIsFree(ctx context.Context, carNum string) (bool, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	row := p.pool.QueryRow(ctxTimeout, "select status from car_park where gos_num=$1", carNum)

	var status bool
	err := row.Scan(&status)
	return status, err
}
