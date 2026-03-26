#!/bin/bash

set +e {0}

 go test -v -coverprofile=coverage.out ./...
 echo "finished tests"