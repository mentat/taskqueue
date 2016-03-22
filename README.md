# taskqueue

A deferred task execution system using RabbitMQ written in Go.  It is a
lightweight replacement for Celery (in Webhook mode).

Taskqueue listens on a RabbitMQ connection for tasks to be created by the
application.  It then executes a request to the given task URL with a
specified Payload.  

## Building

    go build

## Testing

    go test

## Configuration

See the __taskqueue.ini__ file for details on configuring the service.

## Running

Configure __taskqueue.ini__ as needed and move to the desired location.

Run:

    ./taskqueue

or use supplied init file.

The default location for the configuration file is /etc/taskqueue/taskqueue.ini
but you can also set it via the TASKQUEUE_CONFIG_FILE environmental variable.
