# aws-probe

[![Go Report Card](https://goreportcard.com/badge/github.com/xenos76/aws-probe)](https://goreportcard.com/report/github.com/xenos76/aws-probe)

<p align="center">
    <img width="650" alt="aws-probe Logo" src="./assets/img/aws-probe-logo.png"/><br />
    <i>A tool for checking cloud wiring</i>
</p>

`aws-probe` is a CLI toolkit designed to troubleshoot and verify connectivity to AWS resources. It helps developers and operators confirm that their "wiring" (IAM roles, security groups, network paths) is correctly configured for various AWS services.

## Features

- **Identity Verification**: Quickly check your current AWS credentials and assumed role.
- **Service Inspection**: List and probe resources for S3, SQS, SNS, and Secrets Manager.
- **MSK Power Tools**: List MSK clusters and topics, and interact with Kafka via IAM authentication to produce and consume messages.

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

Note: `aws-probe` is not configured and signed as a MacOS app. Manual
steps might be needed to enable the execution of the binary.

</details>

## Usage

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

### SNS & SQS

```shell
# List SNS topics
aws-probe sns --list-topics

# List SNS subscriptions for a topic
aws-probe sns --list-subscriptions <topic-arn>

# List SQS queues
aws-probe sqs --list-queues
```

## License

[MIT License](LICENSE)
