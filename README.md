# datadog-golang-example

An example Go application demonstrating how to instrument services with Datadog APM (tracing) and DogStatsD (metrics). This repo contains minimal, focused examples to show how to add observability to a Go service and how to run it locally (including with a Datadog Agent).

> NOTE: This README contains placeholders and examples. Adjust environment variables, service names, versions, and commands to match the code in this repository.

## Table of contents

- [Overview](#overview)
- [Features](#features)
- [Prerequisites](#prerequisites)
- [Quickstart (local)](#quickstart-local)
- [Running with Docker and Datadog Agent](#running-with-docker-and-datadog-agent)
- [Configuration](#configuration)
- [Instrumentation examples](#instrumentation-examples)
  - [Tracing (APM) — ddtrace-go](#tracing-apm---ddtrace-go)
  - [Metrics — DogStatsD (statsd)](#metrics---dogstatsd-statsd)
- [Build and test](#build-and-test)
- [Contributing](#contributing)
- [License](#license)
- [Contact](#contact)

## Overview

This repository provides examples to help instrument Go services for Datadog observability:
- Add distributed tracing with Datadog APM (ddtrace-go).
- Emit application metrics with DogStatsD (datadog/statsd).
- Example Docker Compose setup to run a local Datadog Agent for development.

The code is intentionally minimal so you can quickly see how to add instrumentation to handlers, background jobs, and libraries.

## Features

- HTTP server instrumented for traces
- Example of starting and finishing spans around operations
- Sending counters, gauges, and histograms to DogStatsD
- Docker Compose file (example) to run a Datadog Agent locally

## Prerequisites

- Go 1.20+ (adjust as needed)
- (Optional) Docker & docker-compose for running a local Datadog Agent
- Datadog account if you want to send data to the cloud (API key required for the cloud Agent)

## Quickstart (local)

1. Clone the repository:
   ```bash
   git clone https://github.com/i3onilha/datadog-golang-example.git
   cd datadog-golang-example
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Set environment variables to point tracing/metrics at your local Agent (default DogStatsD/Trace Agent ports shown):
   ```bash
   export DD_TRACE_AGENT_HOSTNAME=localhost
   export DD_TRACE_AGENT_PORT=8126
   export DD_AGENT_HOST=localhost
   export DD_DOGSTATSD_PORT=8125
   export DD_ENV=development
   export DD_SERVICE=datadog-golang-example
   export DD_VERSION=0.1.0
   ```

4. Run the example service:
   ```bash
   go run ./main.go
   ```

5. Send a request (example):
   ```bash
   curl http://localhost:8080/ping
   ```

Then check your local Datadog Agent UI or Datadog dashboard for traces/metrics.

## Running with Docker and Datadog Agent (example)

You can run a local Datadog Agent with Docker Compose for development. Below is an example `docker-compose.yml` snippet you can add or adapt:

```yaml
version: "3.7"
services:
  datadog:
    image: "gcr.io/datadoghq/agent:latest"
    environment:
      - DD_API_KEY=${DD_API_KEY:-your_api_key_here}
      - DD_APM_ENABLED=true
      - DD_LOGS_ENABLED=false
      - DD_DOGSTATSD_PORT=8125
    ports:
      - "8126:8126"   # Trace agent
      - "8125:8125/udp" # DogStatsD
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
```

Start the agent:
```bash
export DD_API_KEY=your_api_key_here
docker-compose up -d datadog
```

Then run the Go service locally (with env variables pointing to localhost) and it will send metrics/traces to the local agent.

## Configuration

Common environment variables used for Datadog instrumentation:

- DD_AGENT_HOST / DD_TRACE_AGENT_HOSTNAME: host where the Datadog Agent runs (default: localhost)
- DD_TRACE_AGENT_PORT: port for the APM Trace Agent (default: 8126)
- DD_DOGSTATSD_PORT: DogStatsD port (default: 8125)
- DD_ENV: runtime environment (development, staging, production)
- DD_SERVICE: logical service name
- DD_VERSION: service version
- DD_API_KEY: Datadog API key (only needed for Agent to send to Datadog if you run the Agent)

Adjust these variables to fit your environment or CI.

## Instrumentation examples

Below are short examples showing how to use the common Datadog Go libraries. Replace imports and function names to match your code.

### Tracing (APM) — ddtrace-go

Install:
```bash
go get gopkg.in/DataDog/dd-trace-go.v1/ddtrace
go get gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer
```

Initialize tracer early in your app (e.g., main):
```go
import (
  "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func main() {
  tracer.Start(
    tracer.WithAgentAddr("localhost:8126"),
    tracer.WithServiceName("datadog-golang-example"),
    tracer.WithEnv("development"),
  )
  defer tracer.Stop()

  // run server...
}
```

Create spans around handlers/operations:
```go
span, ctx := tracer.StartSpanFromContext(ctx, "handler.process")
def defer span.Finish()

// add tags
span.SetTag("user.id", 123)
```

Instrument HTTP server handlers (example using net/http):
```go
import "gopkg.in/DataDog/dd-trace-go.v1/contrib/net/http"

mux := http.NewServeMux()
// your handler registrations
wrapped := httptrace.WrapHandler(mux, "web.handler")
http.ListenAndServe(":8080", wrapped)
```

(Adjust imports for the dd-trace-go contrib wrappers you use.)

### Metrics — DogStatsD (statsd)

Install:
```bash
go get github.com/DataDog/datadog-go/statsd
```

Create a client and send metrics:
```go
import "github.com/DataDog/datadog-go/statsd"

client, _ := statsd.New("127.0.0.1:8125")
def defer client.Close()

client.Count("example.request.count", 1, []string{"handler:hello"}, 1)
client.Gauge("example.request.latency", 123.45, nil, 1)
client.Histogram("example.processing.time", 250, nil, 1)
```

Use tags liberally (service, env, endpoint, status) to filter and aggregate metrics in Datadog.

## Build and test

Build:
```bash
go build ./...
```

Run unit tests:
```bash
go test ./... -v
```

If tests require a running Agent or specific env vars, set them in CI or your local environment.

## Contributing

Contributions are welcome. Suggested workflow:
1. Fork the repo
2. Create a feature branch
3. Add code / tests / docs
4. Open a pull request describing changes

Please include unit tests and update README or examples when adding features.

## License

Specify a license for the repository (e.g., MIT). If no license file exists, add one.

## Contact

Created by i3onilha. For questions or help, open an issue in this repository.
