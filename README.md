# Golang REST server framework
This will contain basic functionality to setup REST server. Features
including basic server configuration using `echo` framework, which support request
parameters and handler chaining. Logging uses by default `logrus` logger
with structured logging. Logger will by default output JSON logs to STDOUT.
There is also configuration support so that configuration file can be read.

## Service contracts
 Our service contract states, that example `/healthz` call should not be logged
 to keep log amounts in reasonable. Following way is suggested way to attach
 routers to engine:
```golang
	router := echo.New()
	router.Use(
		middleware.RecoverWithConfig(middleware.RecoverConfig{
        			StackSize:  1 << 10, // 1 KB
        		}),
	)

	router.GET("/swagger.json", swagger)
	router.GET("/healthz", healthz)
	router.GET("/version", version)

	traced := router.Group("/")
	traced.Use(
		rest.RequestLogger,
		rest.Recovery,
		rest.RequestTracer,
	)

	// Attach normal routers to traced router group.
```

## Framework
 HTTP layer is handled by `github.com/labstack/echo`. There is default handlers
 injected to chain, which will handle request logging and initializing the request.

### Request initialization
 Request initialization will add `trID` to request context if it is not defined
 in request header. It will also add timeout to context, which should be used all
 long running operations to cancel operations which take too long. There is no point
 to run long tasks, when client will any way timeout after about 60 second.

### Request logging
 Request object will contain context, which will include logger. Logger is predefined
 with `trID` field, which will be part of log entries if that logger instance is used.
 This should be used to make it easier to track log entries, which are initialized by
 one request to server. There is request logger handler injected to handler chain, so
 that it will log all request. Currently it request then when they finish, if it also
 needed to log when request arrive it is possible.

### Request tracing
 There is `RequestTracer()` middleware handler, which can be used to
 extract opentracing data from request headers. It adds span data to request
 context and new span can be initialized using that data by using
 `opentracing.StartSpanFromContext(ctx, "operation_name")`. Environment variable
 `JAEGER_HOST` can be used to give remote endpoint, which will collect tracing data.

### Pprof profiling
 To add `pprof` profiling entries to HTTP server use `rest.AddPprof(router)`, where
 `router` is used the echo router. After that is added `pprof` tools can be used to profile
 running server.

### Tests
 Tests can be run from command line `go test -v -race ./...` or using Docker image.
 option `-v` is optional it puts verbose output of which test is running etc. Option
 `-race` should be used allways. It detects race conditions, which would be
 good to catch in development phase if possible.

## Use of repository

### Bitbucket and private repositories
Due to fact that this is private repository and that Go will use http as default
protocol to fetch modules, when `go get` is used. It is needed (at least in go 1.8
and earlier) to run following configuration
`git config --global url."git@bitbucket.org:".insteadOf "https://bitbucket.org/"`
 which will cause it to use ssh method instead of https and that works, when
 ssh finds authentication key.