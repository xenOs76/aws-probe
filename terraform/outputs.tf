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
