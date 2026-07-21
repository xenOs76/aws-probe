{
  pkgs,
  lib,
  config,
  inputs,
  ...
}: {
  # https://devenv.sh/basics/
  env = {
    GREET = "devenv";
    AWS_ACCESS_KEY_ID = "test";
    AWS_SECRET_ACCESS_KEY = "test";
    AWS_DEFAULT_REGION = "us-east-1";
    AWS_ENDPOINT_URL = "http://localhost:4566";
  };

  # https://devenv.sh/packages/
  packages = with pkgs; [
    docker-compose
    git
    opentofu
    awscli2
    jq
  ];

  # https://devenv.sh/languages/
  # languages.go.enable = true; # Prefer host/toolchain Go (newer than devenv overlay)


  # https://devenv.sh/processes/
  processes.ministack.exec = "docker compose up";

  # https://devenv.sh/services/
  # services.postgres.enable = true;

  # https://devenv.sh/scripts/
  scripts = {
    "iam-list-roles".exec = ''
      set -eu
      ENDPOINT="http://localhost:4566"
      aws --endpoint-url="$ENDPOINT" --region us-east-1 iam list-roles \
        --query 'Roles[?starts_with(RoleName, `aws-probe-`)].RoleName'
    '';

    "iam-list-role-policies".exec = ''
      set -eu
      ENDPOINT="http://localhost:4566"
      ROLE_NAME="''${1:-aws-probe-s3-read-only}"
      aws --endpoint-url="$ENDPOINT" --region us-east-1 iam list-attached-role-policies \
        --role-name "$ROLE_NAME"
    '';

    "iam-show-s3-read-only-policy".exec = ''
      set -eu
      cd "$DEVENV_ROOT/terraform"
      ENDPOINT="http://localhost:4566"
      aws --endpoint-url="$ENDPOINT" --region us-east-1 iam get-policy-version \
        --policy-arn "$(tofu output -raw iam_s3_read_only_policy_arn)" \
        --version-id v1
    '';

    "iam-show-secret-1-policy".exec = ''
      set -eu
      cd "$DEVENV_ROOT/terraform"
      ENDPOINT="http://localhost:4566"
      aws --endpoint-url="$ENDPOINT" --region us-east-1 iam get-policy-version \
        --policy-arn "$(tofu output -raw iam_secret_1_read_only_policy_arn)" \
        --version-id v1
    '';

    # Internal: print shell exports for S3 read-only assumed role (stdout only).
    "iam-exports-s3-ro".exec = ''
      set -eu
      cd "$DEVENV_ROOT/terraform"
      ENDPOINT="http://localhost:4566"
      export AWS_ACCESS_KEY_ID="$(tofu output -raw iam_assumer_access_key_id)"
      export AWS_SECRET_ACCESS_KEY="$(tofu output -raw iam_assumer_secret_access_key)"
      unset AWS_SESSION_TOKEN
      CREDS=$(aws --endpoint-url="$ENDPOINT" --region us-east-1 sts assume-role \
        --role-arn "$(tofu output -raw iam_s3_read_only_role_arn)" \
        --role-session-name aws-probe-verify --output json)
      AK=$(echo "$CREDS" | jq -r '.Credentials.AccessKeyId')
      SK=$(echo "$CREDS" | jq -r '.Credentials.SecretAccessKey')
      ST=$(echo "$CREDS" | jq -r '.Credentials.SessionToken')
      printf 'export AWS_ACCESS_KEY_ID=%q\n' "$AK"
      printf 'export AWS_SECRET_ACCESS_KEY=%q\n' "$SK"
      printf 'export AWS_SESSION_TOKEN=%q\n' "$ST"
      printf 'export AWS_ENDPOINT_URL=%q\n' "$ENDPOINT"
      printf 'export AWS_DEFAULT_REGION=%q\n' "us-east-1"
    '';

    # Internal: print shell exports for Ministack root credentials (stdout only).
    "iam-exports-root".exec = ''
      set -eu
      ENDPOINT="http://localhost:4566"
      printf 'unset AWS_SESSION_TOKEN\n'
      printf 'export AWS_ACCESS_KEY_ID=%q\n' "test"
      printf 'export AWS_SECRET_ACCESS_KEY=%q\n' "test"
      printf 'export AWS_ENDPOINT_URL=%q\n' "$ENDPOINT"
      printf 'export AWS_DEFAULT_REGION=%q\n' "us-east-1"
    '';

    # Apply S3 read-only role in a nested shell (aws-vault style).
    "assume-s3-ro".exec = ''
      set -eu
      eval "$(iam-exports-s3-ro)"
      echo "Assumed aws-probe-s3-read-only" >&2
      exec "''${SHELL:-bash}"
    '';

    # Apply Ministack root credentials in a nested shell.
    "assume-root".exec = ''
      set -eu
      eval "$(iam-exports-root)"
      echo "Using Ministack root credentials" >&2
      exec "''${SHELL:-bash}"
    '';

    # Back-compat aliases for the long names.
    "iam-assume-s3-read-only".exec = ''
      set -eu
      assume-s3-ro
    '';

    "iam-assume-root".exec = ''
      set -eu
      assume-root
    '';

    "iam-verify-aws-probe".exec = ''
      set -eu
      eval "$(iam-exports-s3-ro)"
      go run . whoami
      go run . s3 --list-buckets
    '';

    # Unit tests (mocks only; no Ministack required). Extra args are forwarded to go test.
    "test-unit".exec = ''
      set -eu
      echo "Running unit tests..."
      go test -mod=vendor ./... -count=1 "$@"
    '';

    # Integration tests against Ministack + terraform sample stack.
    # Extra args are forwarded to go test (e.g. -v, -run TestIAM).
    "test-integration".exec = ''
      set -eu
      ENDPOINT="''${AWS_ENDPOINT_URL:-http://localhost:4566}"
      if ! curl -sf "$ENDPOINT/_ministack/health" >/dev/null 2>&1; then
        echo "error: Ministack not healthy at $ENDPOINT/_ministack/health" >&2
        echo "start it with: docker compose up -d  (or devenv up)" >&2
        echo "then apply sample resources: ./setup-local-env.sh" >&2
        exit 1
      fi
      echo "Ministack healthy at $ENDPOINT — running integration tests..."
      go test -mod=vendor -tags=integration ./internal/integration/ -count=1 -timeout 120s "$@"
    '';

    # Unit tests, then integration if Ministack is up (same behavior as devenv test).
    "test-all".exec = ''
      set -eu
      test-unit
      ENDPOINT="''${AWS_ENDPOINT_URL:-http://localhost:4566}"
      if curl -sf "$ENDPOINT/_ministack/health" >/dev/null 2>&1; then
        test-integration
      else
        echo "Ministack not reachable at $ENDPOINT — skipping integration tests"
        echo "(run test-integration after ./setup-local-env.sh to force them)"
      fi
    '';
  };

  # https://devenv.sh/basics/
  enterShell = ''
    echo "IAM verification: assume-s3-ro then aws sts get-caller-identity"
    echo "Reset to root: assume-root"
    echo "Other scripts: iam-list-roles, iam-list-role-policies, iam-show-s3-read-only-policy, iam-show-secret-1-policy, iam-verify-aws-probe"
    echo "Tests: test-unit | test-integration (needs Ministack) | test-all"
  '';

  # https://devenv.sh/tasks/
  # tasks = {
  #   "myproj:setup".exec = "mytool build";
  #   "devenv:enterShell".after = [ "myproj:setup" ];
  # };

  # https://devenv.sh/tests/
  enterTest = ''
    set -eu
    test-all
  '';

  # https://devenv.sh/git-hooks/
  # git-hooks.hooks.shellcheck.enable = true;

  # See full reference at https://devenv.sh/reference/options/
}
