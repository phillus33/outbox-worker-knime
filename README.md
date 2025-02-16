# Outbox Worker KNIME

A Go implementation of the Transactional Outbox pattern using PostgreSQL and NATS, providing at-least-once and FIFO message delivery guarantees.

## Prerequisites

- Go 1.22 or later
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

## Example Application

The example application demonstrates how to use the Outbox Worker with a sample workflow. It showcases the message delivery guarantees and how to interact with the PostgreSQL and NATS components.


## Configuration

The application can be configured using environment variables. Below are the available configuration options:

### Database Configuration

- **DATABASE_URL**: The connection string for the PostgreSQL database. Example: `postgres://user:password@localhost:5432/dbname?sslmode=disable`

### NATS Configuration

- **NATS_URL**: The URL of the NATS server. Example: `nats://localhost:4222`

### Worker Configuration

- **WORKER_POLL_INTERVAL**: The interval for polling for pending messages. Example: `1s`
- **WORKER_BATCH_SIZE**: The number of messages to process in a single batch. Example: `100`
- **WORKER_MAX_RETRIES**: The maximum number of retries for failed messages. Example: `3`





