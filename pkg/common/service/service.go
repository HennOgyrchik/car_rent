package service

import (
	"car_rent/pkg/common/db"
	"car_rent/pkg/common/models"
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4"
	"net/http"
	"strconv"
	"time"
)

type Service struct {
	ctx           context.Context
	db            *db.PSQL
	baseCost      float64
	interval      int
	maxRentPeriod int
}

var (
	ErrInvalidFormatDate = fmt.Errorf("Invalid format date")
	ErrStartLongerStop   = fmt.Errorf("The start date of the lease cannot be longer than the end date")
	ErrWeekdays          = fmt.Errorf("The beginning and end of the lease can only fall on weekdays (Mon-Fri)")
	ErrExceedMaxPeriod   = fmt.Errorf("The allowed rental period has been exceeded")
)

func New(ctx context.Context, db *db.PSQL, baseCost float64, interval int, maxRentPeriod int) *Service {
	return &Service{ctx: ctx, db: db, baseCost: baseCost, interval: interval, maxRentPeriod: maxRentPeriod}
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
		sendError(c, http.StatusBadRequest, fmt.Errorf("The maximum rental period is %d days", s.maxRentPeriod))
		return
	}

	cost := s.calculate(count)

	c.JSON(http.StatusOK, struct {
		Cost float64
	}{Cost: cost})
}
func (s *Service) NewRent(c *gin.Context) {
	rent, period, free, err := s.preparingDataForRent(c)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		sendError(c, http.StatusNotFound, err)
		return
	case errors.Is(err, ErrInvalidFormatDate) || errors.Is(err, ErrStartLongerStop) || errors.Is(err, ErrWeekdays) || errors.Is(err, ErrExceedMaxPeriod):
		sendError(c, http.StatusBadRequest, err)
		return
	case err != nil:
		sendError(c, http.StatusInternalServerError, err)
		return
	case !free:
		sendError(c, http.StatusBadRequest, fmt.Errorf("The car is busy"))
		return
	}

	cost := s.calculate(period)

	err = s.db.NewRent(s.ctx, rent, cost)
	if err != nil {
		sendError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusCreated, struct {
		Cost float64
	}{Cost: cost})

}
func (s *Service) Check(c *gin.Context) {
	_, _, free, err := s.preparingDataForRent(c)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		sendError(c, http.StatusNotFound, err)
		return
	case err != nil:
		sendError(c, http.StatusInternalServerError, err)
		return
	default:
		c.JSON(http.StatusOK, struct {
			Available bool
		}{Available: free})
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

func periodValidate(start, stop string, maxPeriod int) (startDate time.Time, stopDate time.Time, period int, err error) {
	startDate, err = time.Parse(time.DateOnly, start)
	if err != nil {
		err = ErrInvalidFormatDate
		return
	}
	stopDate, err = time.Parse(time.DateOnly, stop)
	if err != nil {
		err = ErrInvalidFormatDate
		return
	}

	if compare := stopDate.Compare(startDate); compare < 1 {
		err = ErrStartLongerStop
		return
	}

	startWeekday := startDate.Weekday()
	stopWeekday := stopDate.Weekday()

	if startWeekday == time.Saturday || startWeekday == time.Sunday || stopWeekday == time.Saturday || stopWeekday == time.Sunday {
		err = ErrWeekdays
		return
	}

	period = int(stopDate.Sub(startDate).Hours() / 24)

	if period > maxPeriod {
		err = ErrExceedMaxPeriod
		return
	}

	return
}

func (s *Service) preparingDataForRent(c *gin.Context) (rent db.Rent, period int, free bool, err error) {
	var userData models.RentRequest
	err = c.BindJSON(&userData)
	if err != nil {
		return
	}

	var start, stop time.Time

	start, stop, period, err = periodValidate(userData.Start, userData.Stop, s.maxRentPeriod)
	if err != nil {
		return
	}
	rent = db.Rent{
		CarNum: userData.CarNumber,
		Start:  start,
		Stop:   stop,
	}
	free, err = s.db.CarIsFree(s.ctx, rent, s.interval)

	return

}
