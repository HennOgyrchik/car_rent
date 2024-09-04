package models

import (
	"github.com/gin-gonic/gin"
)

type RentRequest struct {
	CarNumber string `json:"car_number" binding:"required"`
	Start     string `json:"start" binding:"required"`
	Stop      string `json:"stop" binding:"required"`
}

type Handler interface {
	CostCalculation(c *gin.Context)
	NewRent(c *gin.Context)
	Check(c *gin.Context)
	Report(c *gin.Context)
}
