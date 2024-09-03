package service

import (
	"car_rent/pkg/common/cache"
	"car_rent/pkg/common/db"
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4"
	"net/http"
	"strconv"
)

type Service struct {
	ctx           context.Context
	cache         cache.Cache
	db            *db.PSQL
	baseCost      float64
	interval      int
	maxRentPeriod int
}

func New(ctx context.Context, cache cache.Cache, db *db.PSQL, baseCost float64, interval int, maxRentPeriod int) *Service {
	return &Service{ctx: ctx, cache: cache, db: db, baseCost: baseCost, interval: interval, maxRentPeriod: maxRentPeriod}
}

func (s *Service) Start(errCh chan error) {

	go s.db.Start(s.ctx, errCh)
}

func (s *Service) Stop() {
	s.db.Stop()
}

func (s *Service) CostCalculation(c *gin.Context) {
	count, err := strconv.Atoi(c.Param("count"))
	if err != nil {
		sendError(c, http.StatusBadRequest, err)
		return
	}

	if count > s.maxRentPeriod {
		sendError(c, http.StatusBadRequest, fmt.Errorf("The allowed number of rental days has been exceeded"))
		return
	}

	c.JSON(http.StatusOK, struct {
		Cost float64
	}{Cost: s.calculate(count)})
}
func (s *Service) NewRent(c *gin.Context) {

}
func (s *Service) Check(c *gin.Context) {
	carNum := c.Param("car")

	status, err := s.db.CarIsFree(s.ctx, carNum)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		sendError(c, http.StatusNotFound, err)

	case err != nil:
		sendError(c, http.StatusInternalServerError, err)
	default:
		c.JSON(http.StatusOK, struct {
			Available bool
		}{Available: status})
	}

}
func (s *Service) Report(c *gin.Context) {

}

func (s *Service) calculate(period int) float64 {
	sum := 0.0

	for i := 1; i <= period; i++ {
		switch {
		case i < 5:
			sum += s.baseCost
		case i > 4 && i < 10:
			sum += s.baseCost * 0.95
		case i > 9 && i < 18:
			sum += s.baseCost * 0.9
		case i > 17:
			sum += s.baseCost * 0.85
		}
	}

	return sum
}

func sendError(c *gin.Context, httpCode int, err error) {
	c.JSON(httpCode, struct {
		Error string
	}{Error: err.Error()})
}
