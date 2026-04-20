#!/usr/bin/env bash

set -e

test -d dist || mkdir dist

CGO_ENABLED=0 go build -o dist/aws-probe
