#!/usr/bin/env bash

set -e

test -d dist || mkdir dist

go build -o dist/aws-probe
