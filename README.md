# poc-dblock

This is a simple implementation of a job queue using a PostgresSQL backend. 
The job queue make use of pessimistic locking (with and without skip locking) to ensure that only one worker can claim a job at a time.

## Requirements

- docker
- docker compose
- bats
- go >= 1.22.1

Tested with the following versions:

- docker 25.0.3
- docker compose 2.24.6
- bats 1.10.0
- go 1.22.1

## Quick start

```shell
# Run the test suite
cd test && bats dblock.bats
```


