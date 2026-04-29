#!/usr/bin/env bash

set -e

[[ -d dist ]] || mkdir dist

CGO_ENABLED=0 go build -o dist/aws-probe
