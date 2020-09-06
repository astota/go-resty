package rest

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"net/http/pprof"
)

// AddPprof sets up router config
func AddPprof(e *echo.Echo) {
	r := e.Group("/debug/pprof")

	r.GET("/block", pprofHandler(pprof.Index))
	r.GET("/heap", pprofHandler(pprof.Index))
	r.GET("/profile", pprofHandler(pprof.Profile))
	r.POST("/symbol", pprofHandler(pprof.Symbol))
	r.GET("/symbol", pprofHandler(pprof.Symbol))
	r.GET("/trace", pprofHandler(pprof.Trace))
}

func pprofHandler(h http.HandlerFunc) echo.HandlerFunc {
	handler := http.HandlerFunc(h)
	return func(c echo.Context) (err error) {
		handler.ServeHTTP(c.Response().Writer, c.Request())
		return
	}
}
