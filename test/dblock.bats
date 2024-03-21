#!/usr/bin/env bats

setup() {
  load 'test_helper/bats-support/load'
  load 'test_helper/bats-assert/load'

  # Build and start containers
  docker compose up -d --wait
}

teardown() {
  # Remove containers and volumes
  docker compose down -v
}

check_single_instance() {
  local line=$1
  local count=$(echo "$output" | grep -c "$line")
  if [ "$count" -gt 1 ]; then
    fail "Line '$line' appears more than once in the output"
  fi
}

@test "Create jobs works" {
  run go run ../client/main.go create -status available -payload 501
  assert_success
  assert_output --partial "Creating job with status 'available' and payload '501'"
  assert_output --partial 'Response: {"id":501,"status":"available","payload":"501","timestamp":'
}
#
@test "List all jobs works" {
  run go run ../client/main.go list
  assert_success
  assert_output --partial 'Listing available jobs'
  for i in {1..500}; do
    assert_output --partial "ID: $i, Status: available, Payload: $i, Timestamp:"
  done
}

@test "Claim job works" {
  run go run ../client/main.go claim 1
  assert_success
  assert_output --partial 'Claimed Job - ID: 1, Status: claimed, Payload: 1'
}

@test "Claim job by ID works" {
  run go run ../client/main.go claim-id -id 1
  assert_success
  assert_output --partial 'Successfully claimed job: {"id":1,"status":"claimed","payload":"1"}'
}

@test "Parallel create works" {
  run parallel go run ../client/main.go create -status available -payload {} ::: {501..600}
  assert_success

  for i in {501..600}; do
    # The ID and payload might not be in order, so we need to check for both
    assert_output --partial "Creating job with status 'available' and payload '$i'"
    assert_output --partial "Response: {\"id\":$i,\"status\":\"available\""
    assert_output --partial "\"payload\":\"$i\",\"timestamp\":"
  done
}

@test "Parallel claim works" {
  # Try to claim jobs in parallel 1000 times
  # Without the 'FOR UPDATE SKIP LOCKED LIMIT 1' lock, this test would fail most of the time
  # I was able to claim the same job multiple times in my 4c8t local machine 5 times out of 5
  # Your mileage may vary with your machine


  # Do not fail the test if the parallel command fails
  run parallel --halt soon,fail=1 -N0 go run ../client/main.go claim ::: {1..1000}

  # Check that the job was claimed only once
  for i in {1..500}; do
    check_single_instance "Claimed Job - ID: $i, Status: claimed, Payload: $i"
  done
}

@test "Parallel claim by ID works" {
  # Loop on the first 100 job and try claiming it 1000 times in parallel
  for i in {1..100}; do
    # Without the 'FOR UPDATE' lock, this test fails would fail most of the time
    # I was able to claim the same job multiple times in my 4c8t machine 5 times out of 5
    # Your mileage may vary with your machine

    # Do not fail the test if the parallel command fails
    run parallel --halt soon,fail=1 -N0 go run ../client/main.go claim-id -id $i ::: {1..1000}

    # Check that the job was claimed only once
    check_single_instance "Successfully claimed job: {\"id\":$i,\"status\":\"claimed\",\"payload\":\"$i\"}"
  done
}