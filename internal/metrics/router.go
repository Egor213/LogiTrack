package metrics

import (
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
)

func ConfigureRouter(handler *echo.Echo) {
	handler.GET("/metrics", echoprometheus.NewHandler())
}
