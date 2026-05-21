## 0.2.0 (2026-05-21)

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
