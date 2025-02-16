# Outbox Worker KNIME

A Go implementation of the Transactional Outbox pattern using PostgreSQL and NATS, providing at-least-once and FIFO message delivery guarantees.

## Prerequisites

- Go 1.20 or later
- Docker and Docker Compose
- Make

## Getting Started

1. Clone the repository:

```
git clone https://github.com/phillus33/outbox-worker-knime.git
```

``` cd outbox-worker-knime ```

2. Build the Docker images:

```
make build
```

3. Start the services:

```
make up
```


4. Run the example application:

```
make run
```

## Development
- Build the project: `make build`
- Run tests: `make test`
- Stop and clean up: `make down`


## Architecture

The system implements the Transactional Outbox pattern with the following components:

- PostgreSQL for message storage
- NATS for message publishing
- Leader election to support distributed deployments
- FIFO message delivery guarantee
- At-least-once delivery semantics


## Testing

The project includes integration tests for the outbox pattern. To run the tests:

```
make test
```









