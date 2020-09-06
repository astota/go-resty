package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/astota/go-logging"
	loggertest "github.com/astota/go-logging/loggertest"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func TestRequestLogger(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		status int
	}{
		{"GET OK", http.MethodGet, "/test", http.StatusOK},
		{"GET bad", http.MethodGet, "/test", http.StatusBadRequest},
		{"GET not found", http.MethodGet, "/test1", http.StatusNotFound},
		{"POST OK", http.MethodPost, "/test", http.StatusOK},
		{"POST bad", http.MethodPost, "/test", http.StatusBadRequest},
	}

	for _, tst := range tests {
		t.Run(tst.name, func(t *testing.T) {
			var logger logging.Logger
			testMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {
					logger = logging.GetLogger(c.Request().Context())
					return next(c)
				}
			}

			handler := func(c echo.Context) (err error) {
				logger = logging.GetLogger(c.Request().Context())
				c.String(tst.status, "bar")
				return
			}

			req := createRequest(tst.method, changePath(tst.path))
			callEchoHandler(t, tst.method, handler, testMiddleware, req)

			l, ok := logger.(*loggertest.TestLogger)
			if !ok {
				t.Errorf("Invalid logger type")
				return
			}
			if l.InfoCount != 2 {
				t.Errorf("Info called %d times, expected twice", l.InfoCount)
			}

			if l.TestOutput != "StartingFinished" {
				t.Errorf("Incorrect log lines, expected: '%s', got '%s'", "StartingFinished", l.TestOutput)
			}

			if val, exists := l.Fields["method"]; exists {
				v, _ := val.(string)
				if v != tst.method {
					t.Errorf("incorrect method, expected: '%s', got: '%s'", tst.method, v)
				}
			} else {
				t.Errorf("method is not field in logger")
			}

			if val, exists := l.Fields["status"]; exists {
				v, _ := val.(int)
				if v != tst.status {
					t.Errorf("incorrect status, expected: '%d', got: '%d'", tst.status, v)
				}
			} else {
				t.Errorf("status is not field in logger")
			}

			if _, exists := l.Fields["elapsed_time"]; !exists {
				t.Errorf("elapsed_time is not field in logger")
			}

			if _, exists := l.Fields["path"]; !exists {
				t.Errorf("path is not field in logger")
			}
		})
	}
}

func TestInitRequest(t *testing.T) {
	tests := []struct {
		name    string
		body    map[string]interface{}
		method  string
		headers map[string]string
		err     string
		ip      string
	}{
		{"Too long body, POST", map[string]interface{}{"test": "long body"}, "POST", map[string]string{}, "http: request body too large", "10.10.10.10"},
		{"GET method", nil, "GET", map[string]string{}, "", "10.10.10.10"},
		{"Request ID not given, generates one", nil, "GET", map[string]string{}, "", "10.10.10.10"},
		{"Request ID given", nil, "GET", map[string]string{"BMG-Request-Id": "test-id"}, "", "10.10.10.10"},
		{"UserAgent given", nil, "GET", map[string]string{"User-Agent": "test-user-agent"}, "", "10.10.10.10"},
		{"ApiKey given", nil, "GET", map[string]string{"BMG-Api-Key": "api-key-123"}, "", "10.10.10.10"},
		{"RetailerApiKey given", nil, "GET", map[string]string{"BMG-Retailer-Api-Key": "retailer-api-key-123"}, "", "10.10.10.10"},
		{"X-Forwarded-For with one ip", nil, "GET", map[string]string{"X-Forwarded-For": "123.123.123.123"}, "", "123.123.123.123"},
		{"X-Forwarded-For with multiple ip", nil, "GET", map[string]string{"X-Forwarded-For": "123.123.123.123,234.234.234.234"}, "", "123.123.123.123"},
		{"X-Forwarded-For with multiple ip and private network", nil, "GET", map[string]string{"X-Forwarded-For": "192.168.0.1,123.123.123.123,234.234.234.234"}, "", "123.123.123.123"},
		{"Auth token given", nil, "GET", map[string]string{"BMG-Auth-Token": "auth-123"}, "", "10.10.10.10"},
		{"Organization ID given", nil, "GET", map[string]string{"BMG-Organization-Id": "1000"}, "", "10.10.10.10"},
	}

	conf := NewConfiguration()
	conf.MaximumBodySize = 10

	for _, tst := range tests {
		t.Run(tst.name, func(t *testing.T) {
			u := uuid.New()
			conf.ApplicationName = u.String()
			SetConfiguration(conf)

			var logger logging.Logger
			handler := func(c echo.Context) (err error) {
				if _, err := ioutil.ReadAll(c.Request().Body); err != nil {
					if err.Error() != tst.err {
						t.Errorf("unknown error, expected: '%s', got: '%s'", tst.err, err.Error())
					}
				}

				logger = logging.GetLogger(c.Request().Context())
				if l, ok := logger.(*loggertest.TestLogger); !ok {
					t.Errorf("incorrect logger type")
				} else {
					// Check request method
					if l.Fields["method"] != tst.method {
						t.Errorf("incorrect method in logger, expected: %s, got: %d", tst.method, l.Fields["method"])
					}

					// Check request id (header: BMG-Request-Id)
					if l.Fields["request_id"] != nil {
						if id, ok := tst.headers["BMG-Request-Id"]; ok && id != l.Fields["request_id"] {
							t.Errorf("incorrect request, expected: '%s', got '%s'", l.Fields["request_id"], id)
						}
					} else {
						t.Errorf("request_id missing")
					}

					// Check Basic log fields
					if name, ok := l.Fields["server_name"]; !ok || name == "" {
						t.Errorf("server_name missing")
					}
					if name, ok := l.Fields["progname"]; !ok || name != conf.ApplicationName {
						t.Errorf("invalid application name")
					}
					if name, ok := l.Fields["user_agent"]; !ok || name != tst.headers["User-Agent"] {
						t.Errorf("invalid user_agent")
					}
					if name, _ := l.Fields["api_key"]; name != tst.headers["BMG-Api-Key"] && name != nil {
						t.Errorf("invalid api_key")
					}
					if name, _ := l.Fields["retailer_api_key"]; name != tst.headers["BMG-Retailer-Api-Key"] && name != nil {
						t.Errorf("invalid retailer_api_key")
					}
					if ip, ok := l.Fields["user_ip"]; !ok || ip != tst.ip {
						t.Errorf("invalid user_ip")
					}
					atoken, aexists := tst.headers["BMG-Auth-Token"]
					if token, ok := l.Fields["auth_token"]; (ok || aexists) && token != atoken {
						t.Errorf("invalid auth token")
					}
					rorg, texists := tst.headers["BMG-Organization-Id"]
					if org, ok := l.Fields["organization_id"]; (ok || texists) && org != rorg {
						t.Errorf("invalid organization id")
					}

					// Check DefaultContext
					if fctx, err := GetDefaultContext(c.Request().Context()); err != nil {
						t.Errorf("default context missing")
					} else {
						if fctx.RequestID != l.Fields["request_id"] {
							t.Errorf("context contains invalid request_id")
						}
						if fctx.ForwardedFor != c.Request().Header.Get("X-Forwarded-For") {
							t.Errorf("incorrect X-Forwarded-For in DefaultContext, expected: '%s', got: '%s'", fctx.ForwardedFor, c.Request().Header.Get("X-Forwarded-For"))
						}
						if fctx.OrganizationID != rorg {
							t.Errorf("incorrect OrganizationID in DefaultContext, expected: '%s', got: '%s'", rorg, fctx.OrganizationID)
						}
					}
				}
				c.String(http.StatusOK, "bar")
				return
			}

			opts := []requestOption{addBody(tst.body)}
			for k, v := range tst.headers {
				opts = append(opts, addHeader(k, v))
			}
			req := createRequest(tst.method, opts...)
			callEchoHandler(t, tst.method, handler, nil, req)
		})
	}
}

func timeoutChecking(r *http.Request, d time.Duration) bool {
	ch := make(chan bool)

	go func() {
		time.Sleep(d)
		ch <- true
	}()

	ctx := r.Context()
	select {
	case <-ch:
	case <-ctx.Done():
		return true
	}
	return false
}

func TestInitRequestContextTimeout(t *testing.T) {
	var timeout bool
	var expected = "trIDTest"
	h := make(map[string]string)
	h["trID"] = expected

	conf := NewConfiguration()
	conf.MaximumRequestDuration = 10 * time.Millisecond
	SetConfiguration(conf)
	rr := callHandler(t, http.MethodGet, func(w http.ResponseWriter, r *http.Request) {
		timeout = timeoutChecking(r, (1 * time.Second))
	}, h, nil)

	if !timeout {
		t.Errorf("Function does not get timeout signal")
	}

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

}

func TestInitRequestContextNoTimeout(t *testing.T) {
	var timeout bool
	var expected = "trIDTest"
	h := make(map[string]string)
	h["trID"] = expected
	rr := callHandler(t, http.MethodGet, func(w http.ResponseWriter, r *http.Request) {
		timeout = timeoutChecking(r, (0 * time.Second))
	}, h, nil)

	if timeout {
		t.Errorf("Function does got timeout signal")
	}

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

}

func TestPanicRecovery(t *testing.T) {
	teardownTest := setupTest(t)
	defer teardownTest(t)

	handler := func(c echo.Context) (err error) {
		panic(1)
	}

	app := echo.New()
	app.Use(Recovery)
	app.Logger.SetLevel(99)
	app.GET("/test", handler)
	logger := logging.NewLogger()
	req, _ := http.NewRequest("GET", "/test", nil)
	req = req.WithContext(logging.SetLogger(context.Background(), logger))

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Panic is not revocered correctly")
		}
	}()

	resp := httptest.NewRecorder()
	app.ServeHTTP(resp, req)
}

func setupTest(t *testing.T) func(*testing.T) {
	logging.UseLogger("test-logger")
	loggertest.ResetTestLogger()

	return func(t *testing.T) {
	}
}

func init() {
	loggertest.RegisterTestLogger()
	logging.UseLogger("test-logger")
}

func callHandler(t *testing.T, method string, h http.HandlerFunc, headers map[string]string, body map[string]interface{}) *httptest.ResponseRecorder {
	teardown := setupTest(t)
	defer teardown(t)

	router := http.NewServeMux()
	router.HandleFunc("/test", h)

	opts := []requestOption{addBody(body)}
	for k, v := range headers {
		opts = append(opts, addHeader(k, v))
	}
	return makeRequest(method, router, createRequest(method, opts...))
}

func callEchoHandler(t *testing.T, method string, h echo.HandlerFunc, middleware echo.MiddlewareFunc, req *http.Request) *httptest.ResponseRecorder {
	teardown := setupTest(t)
	defer teardown(t)

	app := echo.New()
	app.Logger.SetLevel(99)
	app.Use(RequestLogger)
	if middleware != nil {
		app.Use(middleware)
	}
	switch method {
	case http.MethodPost:
		app.POST("/test", h)
	case http.MethodGet:
		app.GET("/test", h)
	default:
		t.Errorf("Test does not support method %s", method)
	}

	return makeRequest(method, app, req)
}

type requestOption func(r *http.Request)

func addHeader(key, value string) requestOption {
	return func(r *http.Request) {
		r.Header.Set(key, value)
	}
}

func addBody(body map[string]interface{}) requestOption {
	return func(r *http.Request) {
		b, err := json.Marshal(body)
		if err != nil {
			panic("cannot marshal request body")
		}
		r.Body = ioutil.NopCloser(bytes.NewReader(b))
	}
}

func changePath(path string) requestOption {
	return func(r *http.Request) {
		r.URL, _ = url.Parse(path)
	}
}

func makeRequest(method string, router http.Handler, req *http.Request) *httptest.ResponseRecorder {
	resp := httptest.NewRecorder()
	router = InitRequest(router)
	router.ServeHTTP(resp, req)

	return resp
}

func createRequest(method string, opts ...requestOption) *http.Request {
	req, _ := http.NewRequest(method, "/test", nil)
	req.Host = "localhost"
	req.RemoteAddr = "10.10.10.10:10000"
	for _, opt := range opts {
		opt(req)
	}
	return req
}

func TestGetDefaultContext(t *testing.T) {
	tests := []struct {
		name    string
		want    interface{}
		wantErr bool
	}{
		{"not defined", nil, true},
		{"defined", DefaultContext{OrganizationID: "123"}, false},
		{"corrucpted", "just string", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.want != nil {
				ctx = context.WithValue(ctx, fcKey, tt.want)
			}
			got, err := GetDefaultContext(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDefaultContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			} else if err != nil {

			} else if tt.want != nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetDefaultContext() = %v, want %v", got, tt.want)
			}
		})
	}
}
