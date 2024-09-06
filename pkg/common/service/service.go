package service

import (
	"car_rent/pkg/common/db"
	"car_rent/pkg/common/models"
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4"
	"math"
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
	ErrInvalidFormatDate = fmt.Errorf("invalid format date")
	ErrStartLongerStop   = fmt.Errorf("the start date of the lease must be less than the end date")
	ErrWeekdays          = fmt.Errorf("the beginning and end of the lease can only fall on weekdays (Mon-Fri)")
	ErrExceedMaxPeriod   = fmt.Errorf("the allowed rental period has been exceeded")
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
	dateString := c.Param("date")

	date, err := parseDate(dateString)
	if err != nil {
		sendError(c, http.StatusBadRequest, err)
		return
	}

	r, err := s.db.Report(s.ctx, date)
	if err != nil {
		sendError(c, http.StatusInternalServerError, err)
		return
	}

	sumPercent := 0.0
	for _, v := range r {
		sumPercent += v
	}

	c.JSON(http.StatusOK, models.Report{
		ByCar: r,
		Summary: models.Summary{
			Cars:    len(r),
			Average: math.Round(sumPercent/float64(len(r))*100) / 100,
		},
	})
}

func parseDate(date string) (time.Time, error) {
	result, err := time.Parse(time.DateOnly, date)
	if err != nil {
		err = ErrInvalidFormatDate
	}
	return result, err
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
	startDate, err = parseDate(start)
	if err != nil {
		return
	}
	stopDate, err = parseDate(stop)
	if err != nil {
		return
	}

	if compare := stopDate.Compare(startDate); compare < 0 {
		err = ErrStartLongerStop
		return
	}

	startWeekday := startDate.Weekday()
	stopWeekday := stopDate.Weekday()

	if startWeekday == time.Saturday || startWeekday == time.Sunday || stopWeekday == time.Saturday || stopWeekday == time.Sunday {
		err = ErrWeekdays
		return
	}

	period = int(stopDate.Sub(startDate).Hours()/24) + 1

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
