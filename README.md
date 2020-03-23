
![SVC logo](logo.svg)

# SVC - Worker life-cycle manager

[![Go Report Card](https://goreportcard.com/badge/github.com/voi-go/svc?style=flat-square)](https://goreportcard.com/report/github.com/voi-go/svc)
[![Go Doc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/voi-go/svc)

SVC is a framework that creates a long-running service process, managing the
live-cycle of workers. It comes with "batteries-included" for convenience and
uniformity across services.


## Life-cycle

SVC takes care of starting a main service and shutting it down cleanly. An
application comprises of one or more _Worker_. If zero worker are added, SVC
shuts down immediately.

The life-cycle is:

1. **Initialization** phase (`svc.New`). Each service needs a name, a version.
SVC tries to create a [Zap](https://github.com/uber-go/zap) logger that workers
can make use of. The ideas is to have a consistent structure-logging experience
throughout the service.

2. **Adding workers** (`svc.AddWorker`): Each worker needs a name; names have to
be unique, otherwise SVC shuts down immediately. Workers can optionally
implement the `Healther` interface, in which case SVC can report when all
workers are ready or shutdown the service if a worker reports to be unhealthy.
Adding a worker does not initialize nor run the worker, yet. Creating new
workers **should not block**!

3. **Run** phase (`svc.Run`): Initialized and runs all added workers. Worker get
synchronously initialized in the order they were added (`worker.Init`). If a
worker fails to initialize itself, already initialized workers get terminated
and then entire service is shut down. Initializing a worker **should not block**
the service and should be quick as no deadline is given. After all workers have
been initialized, the workers get asynchronously run (`worker.Run`). Worker's
`Run` **should block**!

4. **Shutdown** phase (`svc.Shutdown`): SVC now waits until either: (i) it
got a _SigInt_, _SigTerm_, or _SigHup_, (ii) an error from a running worker, or
(iii) that all workers have finished successfully. Then it asynchronously
terminates all initialized workers (`worker.Terminate`). Failing to terminate a
worker only logs that error, termination of other workers continues. This phase
has a deadline of 15s by default, thus workers should terminate as quick and
graceful as possible.


## Worker

A worker is a component representing some long-running process. It usually is a
server itself, such as a HTTP or gRPC server. Workers often have a _Controller_.

The life-cycle is:

1. **Instantiation** phase: A worker should get instantiated and then added to
the service via `svc.AddWorker(name, worker)`. Each worker needs to have a
unique name.

2. **Initialization** phase (`worker.Init`): A worker gets initialized and
passed a named logger that it can keep to log throughout its life-time.
Initialization must not block. If a worker fails to get initialized, SVC starts
the shutdown routine.

3. **Run** phase (`worker.Run`): A worker should now execute a long-running
task. When the task ends with an error, SVC immediately shuts down.

4. **Termination** phase (`worker.Terminate`): A worker is asked to terminate
within a given grace period. A wait period may also be provided to delay the
termination of workers whilst an external system is refreshing their service
target list. The grace period should be set as follows:
`wait period + max in flight service request period + some buffer`. The `svc`
grace period must always be greater than the wait period. When running in
kubernetes the `terminationGracePeriodSeconds` must be greater than the `svc`
grace period.


## Controller

A controller is the core of a worker, usually containing business logic, in case
of a server worker, usually the router handlers.


## Batteries-included

All added router endpoints are served over HTTP using `WithHTTPServer` option.


### Health checks (`WithHealthz`)

`GET /live` is always returning 200 from the time the service started. This is
to have a point from which it is easy to know that the process is live in the
container.

`GET /ready` is returning 200 if all the ready checks are looking good the
workers. Otherwise it will return 503 with a JSON body of a list of the errors.
This should ideally not be exported since the errors might contain sensitive
information to debug from.


### Metrics (`WithMetrics` & `WithMetricsHandler`)

`GET /metrics` serves all registered Prometheus metrics.

See [Prometheus' http handler](https://godoc.org/github.com/prometheus/client_golang/prometheus/promhttp#Handler).


### Dynamic log level (`WithLogLevelHandlers`)

`GET /loglevel` gets the current log level.

`PUT /loglevel` sets a new log level. This can be useful to temporarily change
the service's log level to `debug` to allow for better troubleshooting.

See [Zap's http_handler.go](https://github.com/uber-go/zap/blob/master/http_handler.go).


### Pprof (Performance profiler) (`WithPProfHandlers`)

`GET /debug/pprof` serves an index page to allow dynamic profiling while the
service is running.

See [net/http/pprof](https://godoc.org/net/http/pprof).


## Usage

```go
package main

import (
	"github.com/voi-go/svc"
	"go.uber.org/zap"
)

var _ svc.Worker = (*dummyWorker)(nil)

type dummyWorker struct{}

func (d *dummyWorker) Init(*zap.Logger) error { return nil }
func (d *dummyWorker) Terminate() error       { return nil }
func (d *dummyWorker) Run() error             { select {} }

func main() {
	s, err := svc.New("minimal-service", "1.0.0")
	svc.MustInit(s, err)

	w := &dummyWorker{}
	s.AddWorker("dummy-worker", w)

	s.Run()
}

```

For more details, see the examples.

### Examples

- [minimal](./examples/minimal/main.go): `go run ./examples/minimal`

## Configuration

### Customization
The framework supports customization by using the options pattern. All customization options should be defined in `options.go`

### Logging
The log format can be configured by providing an `Option` on initialization. The supported formats are:
- JSON `WithDevelopmentLogger()` (default) or `WithProductionLogger()`
- Stackdriver `WithStackdriverLogger()` (prefered if running in GCP)
- Console `WithConsoleLogger()` (use when running locally)
- Customized `WithLogger()` (bring your own format)

## Contributions

We encourage and support an active, healthy community of contributors &mdash;
including you! Details are in the [contribution guide](CONTRIBUTING.md) and
the [code of conduct](CODE_OF_CONDUCT.md). The `svc` maintainers keep an eye on
issues and pull requests, but you can also report any negative conduct to
opensource@voiapp.io.

### Contributors

- [@djui](https://github.com/djui)
- [@drpytho](https://github.com/drpytho)
- [@cvik](https://github.com/cvik)
- [@K-Phoen](https://github.com/K-Phoen)
- [@ronanbarrett](https://github.com/ronanbarrett)
- [@zatte](https://github.com/zatte)

#### I am missing?
If you feel you should be on this list, create a PR to add yourself.

## License

Apache 2.0, see [LICENSE.md](LICENSE.md).

