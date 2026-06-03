# aws-probe

[![Go Report Card](https://goreportcard.com/badge/github.com/xenos76/aws-probe)](https://goreportcard.com/report/github.com/xenos76/aws-probe)

<p align="center">
    <img width="650" alt="aws-probe Logo" src="./assets/img/aws-probe-logo.png"/><br />
    <i>A tool for checking cloud wiring</i>
</p>

`aws-probe` is a CLI toolkit designed to troubleshoot and verify connectivity to
AWS resources. It helps developers and operators confirm that their "wiring"
(IAM roles, security groups, network paths) is correctly configured for various
AWS services.

## Features

- **Identity Verification**: Quickly check your current AWS credentials and
  assumed role.
- **Service Inspection**: List and probe resources for S3, SQS, SNS, and Secrets
  Manager.
- **MSK Power Tools**: List MSK clusters and topics, and interact with Kafka via
  IAM authentication to produce and consume messages.

## Installation

<details>
<summary>View Installation Options</summary>

### Go install

```shell
go install github.com/xenos76/aws-probe@latest
```

### Manual download

Release binaries and DEB, RPM, APK packages can be downloaded from the
[repo's releases section](https://github.com/xenOs76/aws-probe/releases).\
Binaries and packages are built for Linux and MacOS, `amd64` and `arm64`.

### APT

Configure the repo the following way:

```shell
echo "deb [trusted=yes] https://repo.os76.xyz/apt stable main" | sudo tee /etc/apt/sources.list.d/os76.list
```

then:

```shell
sudo apt-get update && sudo apt-get install -y aws-probe
```

### YUM

Configure the repo the following way:

```shell
echo '[os76]
name=OS76 Yum Repo
baseurl=https://repo.os76.xyz/yum/$basearch/
enabled=1
gpgcheck=0
repo_gpgcheck=0' | sudo tee /etc/yum.repos.d/os76.repo
```

then:

```shell
sudo yum install aws-probe
```

### Homebrew

Add Os76 Homebrew repository:

```shell
brew tap xenos76/tap
```

Install `aws-probe`:

```shell
brew install --casks aws-probe
```

Note: `aws-probe` is not configured and signed as a MacOS app. Manual steps
might be needed to enable the execution of the binary.

</details>

## Usage

### Shell Completion

Generate a completion script for your shell:

```shell
aws-probe completion bash
aws-probe completion zsh
aws-probe completion fish
aws-probe completion powershell
```

Quick setup examples:

```shell
# Bash (current session)
source <(aws-probe completion bash)

# Zsh (persisted)
mkdir -p "${fpath[1]}"
aws-probe completion zsh > "${fpath[1]}/_aws-probe"
autoload -Uz compinit && compinit
```

### Check Identity

```shell
aws-probe whoami
```

### MSK Operations

List MSK clusters and topics, produce and consume messages.

```shell
# List all clusters
aws-probe msk --list-clusters

# List topics for a cluster
aws-probe msk --list-topics <cluster-arn>

# Produce a message
aws-probe msk --produce --topic <topic> --message "hello world" --cluster-arn <arn>

# Consume messages from the beginning
aws-probe msk --consume --topic <topic> --cluster-arn <arn> --from-beginning
```

### S3 Operations

Manage S3 buckets and objects.

```shell
# List all buckets
aws-probe s3 --list-buckets

# List objects in a bucket
aws-probe s3 --list-bucket my-bucket --path logs/ --recursive

# Get object metadata
aws-probe s3 --get-metadata my-bucket --key my-file.txt
```

### Secrets Manager

List and retrieve secrets.

```shell
# List all secrets
aws-probe secrets --list-secrets

# Get secret value
aws-probe secrets --get-secret-value my-secret-id
```

### CloudFront

Manage CloudFront distributions and associated resources.

```shell
# List certificates of all CloudFront distributions
aws-probe cloudfront --list-certificates
```

### SNS & SQS

```shell
# List SNS topics
aws-probe sns --list-topics

# List SNS subscriptions for a topic
aws-probe sns --list-subscriptions <topic-arn>

# List SQS queues
aws-probe sqs --list-queues

# Get SQS queue URL by queue name
aws-probe sqs --get-queue-url <queue-name>

# Receive SQS messages from queue URL
aws-probe sqs --receive-message <queue-url>
```

### Local SQS Verification (Terraform)

After bringing up the local stack (`./setup-local-env.sh`), use these commands
to verify the Terraform sample queues and the new SQS CLI features. The
Terraform apply seeds each sample queue with test messages:

```shell
# Confirm queues exist in LocalStack
aws --endpoint-url=http://localhost:4566 --region us-east-1 sqs list-queues

# Verify aws-probe queue listing
aws-probe sqs --list-queues

# Resolve queue URL from queue name
aws-probe sqs --get-queue-url sample-queue-1

# Receive from queue URL (use URL from list/get output)
aws-probe sqs --receive-message <queue-url>
```

### MCP server

Expose aws-probe to AI clients over stdin/stdout
([Model Context Protocol](https://modelcontextprotocol.io/)):

```shell
aws-probe mcp
```

#### Cursor configuration example

```json
{
  "mcpServers": {
    "aws-probe": {
      "command": "aws-probe",
      "args": ["mcp"]
    }
  }
}
```

#### MCP tools (Phase 1)

| Tool                                     | Purpose                           |
| ---------------------------------------- | --------------------------------- |
| `aws_probe_whoami`                       | Caller identity and auth method   |
| `aws_probe_s3_list_buckets`              | List S3 buckets                   |
| `aws_probe_s3_list_objects`              | List objects/prefixes in a bucket |
| `aws_probe_s3_get_object_metadata`       | HeadObject metadata               |
| `aws_probe_sqs_list_queues`              | List SQS queue URLs               |
| `aws_probe_sqs_get_queue_url`            | Resolve queue name to URL         |
| `aws_probe_sqs_receive_message`          | Receive messages (batch capped)   |
| `aws_probe_secrets_list`                 | List secret names/ARNs            |
| `aws_probe_sns_list_topics`              | List SNS topic ARNs               |
| `aws_probe_msk_list_clusters`            | List MSK clusters                 |
| `aws_probe_msk_list_topics`              | List topics for a cluster         |
| `aws_probe_msk_consume`                  | Bounded Kafka consume             |
| `aws_probe_cloudfront_list_certificates` | CloudFront TLS certificate report |

#### MCP prompts

| Prompt                                    | Purpose                                              |
| ----------------------------------------- | ---------------------------------------------------- |
| `aws_probe_prompt_check_credentials`      | Workflow to verify credentials via `aws_probe_whoami` |
| `aws_probe_prompt_audit_s3_prefix`        | Audit/list workflow for S3 bucket prefix               |
| `aws_probe_prompt_cloudfront_cert_report` | CloudFront TLS/cert review workflow                  |

Resources: `aws-probe://docs/agents`, `aws-probe://docs/cli-reference`,
`aws-probe://examples/ministack`.

## License

[MIT License](LICENSE)
