terraform {
  required_version = ">= 1.0"
}

provider "aws" {
  access_key                  = "test"
  secret_key                  = "test"
  region                      = "us-east-1"
  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_requesting_account_id  = true

  endpoints {
    s3             = "http://localhost:4566"
    secretsmanager = "http://localhost:4566"
  }
}

resource "aws_s3_bucket" "test_bucket" {
  bucket = "aws-probe-test-bucket"
}

resource "aws_s3_object" "sample_file_1" {
  bucket       = aws_s3_bucket.test_bucket.id
  key          = "samples/hello.txt"
  content      = "Hello from aws-probe test environment!"
  content_type = "text/plain"
}

resource "aws_s3_object" "sample_file_2" {
  bucket = aws_s3_bucket.test_bucket.id
  key    = "samples/data.json"
  content = jsonencode({
    message   = "This is test data"
    timestamp = "2026-05-05T15:00:00Z"
    items     = ["item1", "item2", "item3"]
  })
  content_type = "application/json"
}

resource "aws_s3_object" "sample_file_3" {
  bucket       = aws_s3_bucket.test_bucket.id
  key          = "samples/nested/deep/test.log"
  content      = "2026-05-05 15:00:00 INFO Test log entry\n2026-05-05 15:01:00 DEBUG Another log entry"
  content_type = "text/plain"
}

resource "aws_secretsmanager_secret" "test_secret_1" {
  name        = "test-secret-1"
  description = "Test secret for aws-probe development"
}

resource "aws_secretsmanager_secret_version" "test_secret_1_value" {
  secret_id     = aws_secretsmanager_secret.test_secret_1.id
  secret_string = "{\"username\":\"testuser\",\"password\":\"testpass123\"}"
}

resource "aws_secretsmanager_secret" "test_secret_2" {
  name        = "test-secret-2"
  description = "Another test secret for aws-probe development"
}

resource "aws_secretsmanager_secret_version" "test_secret_2_value" {
  secret_id     = aws_secretsmanager_secret.test_secret_2.id
  secret_string = "api-key-12345-secret"
}
