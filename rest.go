package rest

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/astota/go-logging"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/sebest/xff"
)

type contextKey string

var fcKey contextKey = "DefaultContext"

// Server configuration
var config = NewConfiguration()

// SetConfiguration sets configuration paremters to REST server
func SetConfiguration(conf Configuration) {
	config = conf
}

// DefaultContext contains Request specific information
type DefaultContext struct {
	RequestID      string
	ForwardedFor   string
	OrganizationID string
}

// InitRequest initializes special variables that we want to use in request
// Handling maximum body size, adds logger, which is attached to request
// and adds timeout handling.
func InitRequest(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			defer r.Body.Close()
			r.Body = http.MaxBytesReader(w, r.Body, config.MaximumBodySize)
			defer r.Body.Close()
		}

		// Extract or generate transaction id
		requestID := ""
		if requestID = r.Header.Get("BMG-Request-Id"); requestID == "" {
			requestID = uuid.New().String()
		}

		userIP := ""
		if sip, _, err := net.SplitHostPort(xff.GetRemoteAddr(r)); err == nil {
			userIP = sip
		}

		logger := logging.NewLogger().AddFields(logging.Fields{
			"request_id":  requestID,
			"server_name": r.Host,
			"progname":    config.ApplicationName,
			"user_agent":  r.Header.Get("User-Agent"),
			"user_ip":     userIP,
		})

		if apiKey := r.Header.Get("BMG-Retailer-Api-Key"); apiKey != "" {
			logger = logger.AddFields(logging.Fields{
				"retailer_api_key": apiKey,
			})
		}

		if apiKey := r.Header.Get("BMG-Api-Key"); apiKey != "" {
			logger = logger.AddFields(logging.Fields{
				"api_key": apiKey,
			})
		}

		if authToken := r.Header.Get("BMG-Auth-Token"); authToken != "" {
			logger = logger.AddFields(logging.Fields{
				"auth_token": authToken,
			})
		}

		var organizationID string
		if oid := r.Header.Get("BMG-Organization-Id"); oid != "" {
			logger = logger.AddFields(logging.Fields{
				"organization_id": oid,
			})
			organizationID = oid
		}

		// Setup context and also add timeout
		ctx, cancel := context.WithTimeout(r.Context(), config.MaximumRequestDuration)
		defer cancel()

		ctx = context.WithValue(ctx, fcKey, DefaultContext{
			RequestID:      requestID,
			ForwardedFor:   r.Header.Get("X-Forwarded-For"),
			OrganizationID: organizationID,
		})

		ctx = logging.SetLogger(ctx, logger)
		r = r.WithContext(ctx)

		// call original handler
		h.ServeHTTP(w, r)
	}
}

func setDefaultContext(ctx context.Context, fctx DefaultContext) context.Context {
	return context.WithValue(ctx, fcKey, fctx)
}

// GetDefaultContext tries to get request specific data from context
func GetDefaultContext(ctx context.Context) (DefaultContext, error) {
	val := ctx.Value(fcKey)
	if val == nil {
		return DefaultContext{}, fmt.Errorf("no DefaultContext")
	}

	if fctx, ok := val.(DefaultContext); ok {
		return fctx, nil
	}

	return DefaultContext{}, fmt.Errorf("default context key is corrupted")
}

// Shutdown will shutdown server gracefully when SIGTERM or SIGINT is received
func shutdown(s *http.Server) {
	// Handle SIGINT and SIGTERM
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT)
	signal.Notify(sig, syscall.SIGTERM)

	<-sig
	logger := logging.NewLogger()
	logger.Info("Shutting down server...")

	// Time which is waited before forcefully shutdown server.
	// Kubernetes default between SIGTERM and SIGKILL
	// is 30s, so shorter time should be configured.
	ctx, cancel := context.WithTimeout(context.Background(), config.ShutdownGraceTime)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		logger.Fatal("Could not shutdown gracefully")
	}
}

// Run starts new server instance and also listen SIGINT and SIGTERM signals
// to gracefully stop server procss.
func Run(s *http.Server) {
	logger := logging.NewLogger()

	// Make custom initializations to request
	s.Handler = InitRequest(s.Handler)

	go shutdown(s)

	// Start server
	if err := s.ListenAndServe(); err != http.ErrServerClosed {
		logger.Fatal(fmt.Sprintf("Server error: %s", err.Error()))
	}

	logger.Info("Server gracefully stopped")
}

// RunTLS starts new server TLS instance and also listen SIGINT and
// SIGTERM signals to gracefully stop server procss.
func RunTLS(s *http.Server) {
	logger := logging.NewLogger()

	// Make custom initializations to request
	s.Handler = InitRequest(s.Handler)

	go shutdown(s)

	// Start server
	if err := s.ListenAndServeTLS("", ""); err != http.ErrServerClosed {
		logger.Fatal(fmt.Sprintf("Server error: %s", err.Error()))
	}

	logger.Info("Server gracefully stopped")
}

// RequestLogger return request logger handler. This will call gin Next()
// method internally, so all other handlers are running inside this.
// Therefore this can be used to log whole request timeline.
func RequestLogger(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()
		logger := logging.GetLogger(req.Context()).AddFields(logging.Fields{
			"method": req.Method,
			"path":   req.URL.Path,
		})
		logger.Info("Starting")
		t := time.Now()

		err := next(c)

		status := c.Response().Status
		if httpError, ok := err.(*echo.HTTPError); ok {
			status = httpError.Code
		}

		// log request
		logger.AddFields(logging.Fields{
			"status":       status,
			"elapsed_time": float64(time.Since(t).Nanoseconds()) / 1000000.0,
		}).Info("Finished")

		return err
	}
}

// Recovery captures panics from handlers and log those
func Recovery(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		defer func() {
			if r := recover(); r != nil {
				st := make([]byte, 1<<15)
				runtime.Stack(st, false)
				httprequest, _ := httputil.DumpRequest(c.Request(), false)
				logger := logging.GetLogger(c.Request().Context())
				logger = logger.AddFields(logging.Fields{
					"stacktrace": string(st),
					"request":    string(httprequest),
				})
				logger.Error("internal server error")
				c.String(http.StatusInternalServerError, "")
			}
		}()
		return next(c)
	}
}
