#!/usr/bin/env bash

# https://sonarcloud.io/project/configuration/GitHubManual?id=xenOs76_aws-probe

sonar-scanner -Dsonar.organization=xenos76 \
  -Dsonar.projectKey=xenOs76_aws-probe \
  -Dsonar.go.coverage.reportPaths=cover.out \
  -Dsonar.exclusions=completions/**,.devenv/**,.direnv/** \
  -D"sonar.tests=." \
  -D"sonar.test.inclusions=*_test.go,**/*_test.go"
