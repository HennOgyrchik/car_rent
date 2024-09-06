package main

import (
	"car_rent/pkg/common/config"
	"car_rent/pkg/common/db"
	"car_rent/pkg/common/service"
	"car_rent/pkg/common/web"
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	defer stop()

	cfg, err := config.Read(ctx)
	if err != nil {
		slog.Error("Load configuration", "error", err.Error())
		return
	}

	webSrv := web.New(cfg.Web.ConnectionURL())

	dbConnURL, err := cfg.Postgres.ConnectionURL()
	if err != nil {
		slog.Error("Prepare DB connection URL", "error", err.Error())
		return
	}

	dbConn := db.New(dbConnURL, time.Duration(cfg.Postgres.ConnTimeout)*time.Second)

	srv := service.New(ctx, dbConn, cfg.Service.BaseCost, cfg.Service.Interval, cfg.Service.MaxRentPeriod)

	errCh := make(chan error)
	defer close(errCh)

	srv.Start(errCh)
	go webSrv.Start(srv, errCh)

	defer func() {
		srv.Stop()
		if err := webSrv.Stop(); err != nil {
			slog.Error("Stop webserver", err)
			return
		}

	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		slog.Error("Start service", err)

	}
}
