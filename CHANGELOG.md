## 0.2.0 (2026-06-04)

### Feat

    Added MCP server functionality to expose aws-probe to AI clients via stdin/stdout using the new mcp CLI command.
    Added eight MCP tools: identity verification, S3 operations, SQS operations, Secrets Manager, SNS, MSK, CloudFront certificate inventory, and bounded MSK message consumption.
    Added MCP prompts for credential checking, S3 auditing, and CloudFront certificate reporting.
    Added MCP resources including CLI reference and agent documentation.

### Doc

    Updated README with MCP server setup guide, tool reference table, and resource links.

## 0.1.3 (2026-05-28)

### Feat

    Added shell completion support for bash, zsh, fish, and powershell shells
    CloudFront command now supports --output and --theme flags for customized output

### Doc

    Updated README with shell completion setup examples and CloudFront usage documentation

### Tests

    Added comprehensive tests for completion commands and command validation logic

## 0.1.2 (2026-05-21)

## CI

- update release Github action with missing Nix dependencies

## 0.1.1 (2026-05-21)

### Feat

- add cloudfront command

## 0.1.0 (2026-05-05)

### Fix

- missing AWS SSO variables
- early fail in cmd test. fix: nil pointer guard in NewService function. fix:
  check on empty S3 listing. fix: missing nil pointer guard in secrets
- naming convention used for interfaces
- possible redirect to http in install script
- duplication issues

### Refactor

- move AWS related code into packages
- split code in internal. Keep the Cobra commands under cmd and move the code
  related to AWS services in dedicated packages. ci: draft a development
  environment simulating an AWS account with Ministack.

## 0.0.4 (2026-04-20)

### Feat

- MSK auth via IAM, produce and consume

## 0.0.3 (2026-04-16)

### Feat

- add list-bucket flag to s3 command

### Fix

- trivy action version

## 0.0.2 (2026-04-12)

### Fix

- missing variable check for static creds auth

### Refactor

- split list command

## 0.0.1 (2026-04-11)

### Feat

- initial import
