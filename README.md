# taskqueue

A deferred task execution system using RabbitMQ written in Go.  It is a
lightweight replacement for Celery (in Webhook mode).

Taskqueue listens on a RabbitMQ connection for tasks to be created by the
application.  It then executes a request to the given task URL with a
specified Payload.  

It works like this:

 1. Some application wants to defer the execution of a task.  To do this
    they create a JSON message and send it to a specific RabbitMQ queue:
```json
{
  "url":"/tasks/my_fun_task",
  "countdown": 20,
  "max_retries": 3,
}
```
 2. The taskqueue service receives this task from RabbitMQ and executes the
    HTTP request after a 20 second wait.  If the result is not a 200, the
    task will retry two more times before failing.

## Message Details

A task execution message has the following format:

 * url - the path or full URL of the task to execute.
 * eta - the Posix timestamp of when this task should run (optional).
 * countdown - the number of seconds to wait before running the task (optional).
 * max_retries - the number of times to retry this task (optional, default is
   to retry forever).
 * payload - the JSON encoded data to send to the function.
 * expires - the Posix timestamp of when this task will expire if not run
   (optional, default is no expiration).
 * queue - the Queue name to send the task to.  Each queue can be configured
   with different concurrency and retry semantics (optional, defaults to
   default queue).
 * metadata - extra JSON data used for request handler extensions (optional).

## Building

    go build

## Testing

    go test

## Benchmark

    go test -bench . -benchtime 10s

## Configuration

See the __taskqueue.ini__ file for details on configuring the service.

## Running

Configure __taskqueue.ini__ as needed and move to the desired location.

Run:

    ./taskqueue

or use supplied init file.

The default location for the configuration file is /etc/taskqueue/taskqueue.ini
but you can also set it via the TASKQUEUE_CONFIG_FILE environmental variable.

For command line options run:

    ./taskqueue -h
