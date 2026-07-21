output "s3_bucket_name" {
  description = "Name of the test S3 bucket"
  value       = aws_s3_bucket.test_bucket.id
}

output "s3_bucket_arn" {
  description = "ARN of the test S3 bucket"
  value       = aws_s3_bucket.test_bucket.arn
}

output "secret_1_name" {
  description = "Name of the first test secret"
  value       = aws_secretsmanager_secret.test_secret_1.name
}

output "secret_1_arn" {
  description = "ARN of the first test secret"
  value       = aws_secretsmanager_secret.test_secret_1.arn
}

output "secret_2_name" {
  description = "Name of the second test secret"
  value       = aws_secretsmanager_secret.test_secret_2.name
}

output "secret_2_arn" {
  description = "ARN of the second test secret"
  value       = aws_secretsmanager_secret.test_secret_2.arn
}

output "sample_queue_1_name" {
  description = "Name of the first sample SQS queue"
  value       = aws_sqs_queue.sample_queue_1.name
}

output "sample_queue_1_url" {
  description = "URL of the first sample SQS queue"
  value       = aws_sqs_queue.sample_queue_1.url
}

output "sample_queue_1_arn" {
  description = "ARN of the first sample SQS queue"
  value       = aws_sqs_queue.sample_queue_1.arn
}

output "sample_queue_2_name" {
  description = "Name of the second sample SQS queue"
  value       = aws_sqs_queue.sample_queue_2.name
}

output "sample_queue_2_url" {
  description = "URL of the second sample SQS queue"
  value       = aws_sqs_queue.sample_queue_2.url
}

output "sample_queue_2_arn" {
  description = "ARN of the second sample SQS queue"
  value       = aws_sqs_queue.sample_queue_2.arn
}

output "iam_s3_read_only_role_name" {
  description = "Name of the S3 read-only IAM role"
  value       = aws_iam_role.s3_read_only.name
}

output "iam_s3_read_only_role_arn" {
  description = "ARN of the S3 read-only IAM role"
  value       = aws_iam_role.s3_read_only.arn
}

output "iam_s3_read_write_role_name" {
  description = "Name of the S3 read-write IAM role"
  value       = aws_iam_role.s3_read_write.name
}

output "iam_s3_read_write_role_arn" {
  description = "ARN of the S3 read-write IAM role"
  value       = aws_iam_role.s3_read_write.arn
}

output "iam_secret_1_read_only_role_name" {
  description = "Name of the secret-1 read-only IAM role"
  value       = aws_iam_role.secret_1_read_only.name
}

output "iam_secret_1_read_only_role_arn" {
  description = "ARN of the secret-1 read-only IAM role"
  value       = aws_iam_role.secret_1_read_only.arn
}

output "iam_secret_2_read_only_role_name" {
  description = "Name of the secret-2 read-only IAM role"
  value       = aws_iam_role.secret_2_read_only.name
}

output "iam_secret_2_read_only_role_arn" {
  description = "ARN of the secret-2 read-only IAM role"
  value       = aws_iam_role.secret_2_read_only.arn
}

output "iam_s3_read_only_policy_arn" {
  description = "ARN of the S3 read-only IAM policy"
  value       = aws_iam_policy.s3_read_only.arn
}

output "iam_s3_read_write_policy_arn" {
  description = "ARN of the S3 read-write IAM policy"
  value       = aws_iam_policy.s3_read_write.arn
}

output "iam_secret_1_read_only_policy_arn" {
  description = "ARN of the secret-1 read-only IAM policy"
  value       = aws_iam_policy.secret_1_read_only.arn
}

output "iam_secret_2_read_only_policy_arn" {
  description = "ARN of the secret-2 read-only IAM policy"
  value       = aws_iam_policy.secret_2_read_only.arn
}

output "iam_assumer_user_name" {
  description = "Name of the IAM user that can assume sample roles"
  value       = aws_iam_user.aws_probe_assumer.name
}

output "iam_assumer_access_key_id" {
  description = "Access key ID for the aws-probe-assumer IAM user"
  value       = aws_iam_access_key.aws_probe_assumer.id
}

output "iam_assumer_secret_access_key" {
  description = "Secret access key for the aws-probe-assumer IAM user"
  value       = aws_iam_access_key.aws_probe_assumer.secret
  sensitive   = true
}
