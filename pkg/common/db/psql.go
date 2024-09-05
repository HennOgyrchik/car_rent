package db

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	"math"
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

	row := p.pool.QueryRow(ctxTimeout, "select count(*) from rent where gos_num = $1 and (period && $2)",
		rent.CarNum,
		fmt.Sprintf("[%s,%s]", rent.Start.Add(-1*time.Duration(serviceDays)*time.Hour*24).Format(time.DateOnly), rent.Stop.Add(time.Duration(serviceDays)*time.Hour*24).Format(time.DateOnly)))

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

	_, err := p.pool.Exec(ctxTimeout, "insert into rent (gos_num, period ,cost) values($1,$2,$3)", rent.CarNum, fmt.Sprintf("[%s,%s]", rent.Start.Format(time.DateOnly), rent.Stop.Format(time.DateOnly)), cost)

	return err
}

func (p *PSQL) Report(ctx context.Context, date time.Time) (report map[string]float64, err error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	y, m, _ := date.Date()

	from := time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(y, m+1, 0, 0, 0, 0, 0, time.UTC)

	rows, err := p.pool.Query(ctxTimeout, "select * from report($1::daterange)", fmt.Sprintf("[%s,%s]", from.Format(time.DateOnly), to.Format(time.DateOnly)))
	if err != nil {
		return
	}

	report = make(map[string]float64)

	for rows.Next() {
		var carNum string
		var count int
		if err = rows.Scan(&carNum, &count); err != nil {
			return
		}
		report[carNum] = math.Round(float64(count)/float64(to.Day())*10000) / 100
	}
	return
}
