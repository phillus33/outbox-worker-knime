# Architecture

## Components

### Store
- Handles message persistence in PostgreSQL
- Ensures FIFO ordering through sequence numbers
- Manages message status transitions

### Worker
- Polls for pending messages
- Ensures ordered processing
- Handles leader election
- Provides at-least-once delivery

### Message Flow
1. Application begins transaction
2. Message stored in outbox table
3. Transaction commits
4. Worker polls for messages
5. Messages published to NATS in sequence

## Configuration

The application can be configured using environment variables. Below are the available configuration options:

### Database Configuration
- **DATABASE_URL**: The connection string for the PostgreSQL database. Example: `postgres://user:password@localhost:5432/dbname?sslmode=disable`
- **DATABASE_MAX_CONNECTIONS**: The maximum number of connections to the database. Default is `10`.

### NATS Configuration
- **NATS_URL**: The URL for the NATS server. Example: `nats://localhost:4222`

### Worker Configuration
- **WORKER_POLL_INTERVAL**: The interval at which the worker polls for new messages. Default is `1s`.
- **WORKER_BATCH_SIZE**: The number of messages to process in a single batch. Default is `100`.
- **WORKER_MAX_RETRIES**: The maximum number of retries for message delivery. Default is `3`.

### Instance Configuration
- **INSTANCE_ID**: A unique identifier for each instance of the application. This is useful for distinguishing between multiple instances running in parallel.

## Deployment

The application can be deployed using Docker and Docker Compose. Below are the steps and considerations for deployment:

### Prerequisites
- Ensure that Docker and Docker Compose are installed on your machine.
- A PostgreSQL database should be accessible, either locally or in a cloud environment.

### Running the Application
1. Clone the repository:
   ```bash
   git clone https://github.com/phillus33/outbox-worker-knime.git
   cd outbox-worker-knime
   ```

2. Build and run the application using Docker Compose:
   ```bash
   docker-compose up --build
   ```

3. The application will start the following services:
   - **PostgreSQL**: For message persistence.
   - **NATS**: For message queuing.
   - **App Instances**: Multiple instances of the application can be run for load balancing.

### Environment Variables
- Ensure that the necessary environment variables are set in your `.env` file or directly in the Docker Compose file.

### Scaling
- You can scale the application by increasing the number of replicas in the Docker Compose file. For example:
  ```yaml
  app1:
    deploy:
      replicas: 3
  ```

### Monitoring
- For future reference, we could use Prometheus and Grafana to monitor the application. Metrics could be:
  - Number of messages in the outbox
  - Number of messages published to NATS
  - Number of messages failed to be published to NATS
  - Number of messages published to NATS in sequence
  - Number of messages published to NATS out of sequence
  - Number of messages published to NATS in a single batch