### 1.0.5

- [FIX] Use correct http status in logs and tracing tags

### 1.0.4

- [ENHANCEMENT] Tests refactore do be more easily extended

### 1.0.3

- [FIX] Tracer middleware to return error from `next(c)` to handle example missing routes correctly.

### 1.0.2

 - [ENHANCEMENT] Added a new function `InitJaegerTracer`: init a Jaeger logger and tracer and returns the closer function. Panic if initialization is not possible [RAD-649]
