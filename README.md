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

List clusters and topics, or produce/consume messages using IAM authentication:

```shell
# List clusters
aws-probe msk list-clusters

# Produce a message with a key
aws-probe msk produce --topic my-topic --cluster-arn <ARN> --message "hello" --key "my-key"

# Consume from the beginning
aws-probe msk consume --topic my-topic --cluster-arn <ARN> --from-beginning
```

### S3 & Other Services

```shell
# List S3 buckets
aws-probe s3 list

# List SQS queues
aws-probe sqs list

# List Secrets
aws-probe secrets list
```

## License

[MIT License](LICENSE)
