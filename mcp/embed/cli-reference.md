# aws-probe CLI reference (summary)

## Global

- Credentials: standard AWS environment, profiles, SSO, EC2 role, EKS IRSA.
- Version: `aws-probe --version`

## Commands

| Command | Actions |
|---------|---------|
| `whoami` | Caller identity and auth method |
| `s3` | `--list-buckets`, `--list-bucket`, `--get-metadata` |
| `sqs` | `--list-queues`, `--get-queue-url`, `--receive-message` |
| `secrets` | `--list-secrets`, `--get-secret-value` |
| `sns` | `--list-topics` |
| `msk` | `--list-clusters`, `--list-topics`, `--produce`, `--consume` |
| `cloudfront` | `--list-certificates` (`--output`, `--theme`) |
| `mcp` | MCP server on stdio |
| `completion` | Shell completions |

## MCP tools (prefix `aws_probe_`)

See the MCP tool list registered by `aws-probe mcp` or kubectl-netdrill with `--external-tools`.
