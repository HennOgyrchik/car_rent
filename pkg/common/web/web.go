package web

import (
	"car_rent/pkg/common/models"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

type Web struct {
	connURL string
	server  *http.Server
}

func New(url string) *Web {
	return &Web{connURL: url}
}

func (w *Web) Start(hndlr models.Handler, errCh chan error) {
	router := gin.Default()
	//gin.SetMode(gin.ReleaseMode)

	router.GET("/rent/calculate/:count", hndlr.CostCalculation) //рассчитать стоимость
	router.POST("/rent", hndlr.NewRent)                         //создать аренду
	router.PUT("/rent/check", hndlr.Check)                      //проверить доступность | PUT не подходит, но без этого не читает JSON
	router.PUT("/rent/report", hndlr.Report)                    //отчет

	w.server = &http.Server{Addr: w.connURL, Handler: router.Handler()}

	if err := w.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		errCh <- fmt.Errorf("Start web server: %w", err)
	}

}

func (w *Web) Stop() error {
	ctxSrv, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var err error
	if err = w.server.Shutdown(ctxSrv); err != nil {
		err = fmt.Errorf("Web server was shutdown incorrectly: %w", err)
	}
	return err
}
