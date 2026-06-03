# aws-probe

`aws-probe` is a CLI program written in Golang with the purpose of getting easy
access to AWS resources in most situations: developer workstation, EC2 instance,
Kubernetes container.

The program uses Cobra and, optionally, Viper for managing the CLI.\
Cobra is the client that uses packages to implement its functions. Service logic
lives under `internal/`; the composable MCP surface is the public `mcp` package.\
The program connects to AWS services using the AWS SDK for Golang V2.\
If required, TUI components are developed using Charmbracelet's libraries like
BubbleTea V2 and LipGloss V2.\
Testing is done creating unit tests: use testify as a base framework. Use
https://github.com/google/go-cmp when comparing complex, deeply nested, structs.
Validation is done using golangci-lint according to the local configuration
file.\
A vulnerability check done with govulncheck completes the list of post update
checks we do.
