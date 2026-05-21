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
    sqs            = "http://localhost:4566"
    cloudfront     = "http://localhost:4566"
    acm            = "http://localhost:4566"
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

resource "aws_sqs_queue" "sample_queue_1" {
  name = "sample-queue-1"
}

resource "aws_sqs_queue" "sample_queue_2" {
  name = "sample-queue-2"
}

resource "terraform_data" "seed_sample_queue_messages" {
  depends_on = [
    aws_sqs_queue.sample_queue_1,
    aws_sqs_queue.sample_queue_2,
  ]

  provisioner "local-exec" {
    command = <<-EOT
      set -eu

      Q1_URL=$(AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test AWS_DEFAULT_REGION=us-east-1 aws --endpoint-url=http://localhost:4566 sqs get-queue-url --queue-name sample-queue-1 --query 'QueueUrl' --output text)
      Q2_URL=$(AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test AWS_DEFAULT_REGION=us-east-1 aws --endpoint-url=http://localhost:4566 sqs get-queue-url --queue-name sample-queue-2 --query 'QueueUrl' --output text)

      AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test AWS_DEFAULT_REGION=us-east-1 aws --endpoint-url=http://localhost:4566 sqs send-message --queue-url "$Q1_URL" --message-body "sample-queue-1 message 1"
      AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test AWS_DEFAULT_REGION=us-east-1 aws --endpoint-url=http://localhost:4566 sqs send-message --queue-url "$Q1_URL" --message-body "sample-queue-1 message 2"
      AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test AWS_DEFAULT_REGION=us-east-1 aws --endpoint-url=http://localhost:4566 sqs send-message --queue-url "$Q2_URL" --message-body "sample-queue-2 message 1"
      AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test AWS_DEFAULT_REGION=us-east-1 aws --endpoint-url=http://localhost:4566 sqs send-message --queue-url "$Q2_URL" --message-body "sample-queue-2 message 2"
    EOT
  }
}

resource "aws_acm_certificate" "cert_1" {
  domain_name       = "example.com"
  validation_method = "DNS"
}

resource "aws_acm_certificate" "cert_2" {
  domain_name       = "test.com"
  validation_method = "DNS"
}

resource "terraform_data" "seed_cloudfront_distributions" {
  depends_on = [
    aws_s3_bucket.test_bucket,
    aws_acm_certificate.cert_1,
    aws_acm_certificate.cert_2
  ]

  provisioner "local-exec" {
    command = <<-EOT
      set -eu

      # Cleanup any old state
      rm -f .cloudfront_dist_ids

      # Dist 1: Default cert
      ID1=$(AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test AWS_DEFAULT_REGION=us-east-1 aws --endpoint-url=http://localhost:4566 cloudfront create-distribution --distribution-config '{"CallerReference":"dist_default_cert","Origins":{"Quantity":1,"Items":[{"Id":"myS3Origin","DomainName":"${aws_s3_bucket.test_bucket.bucket_regional_domain_name}"}]},"DefaultCacheBehavior":{"TargetOriginId":"myS3Origin","ViewerProtocolPolicy":"allow-all"},"ViewerCertificate":{"CloudFrontDefaultCertificate":true},"Comment":"","Enabled":true}' --query 'Distribution.Id' --output text)
      echo "$ID1" >> .cloudfront_dist_ids

      # Dist 2: ACM Cert 1
      ID2=$(AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test AWS_DEFAULT_REGION=us-east-1 aws --endpoint-url=http://localhost:4566 cloudfront create-distribution --distribution-config '{"CallerReference":"dist_acm_1","Origins":{"Quantity":1,"Items":[{"Id":"myS3Origin","DomainName":"${aws_s3_bucket.test_bucket.bucket_regional_domain_name}"}]},"DefaultCacheBehavior":{"TargetOriginId":"myS3Origin","ViewerProtocolPolicy":"https-only"},"ViewerCertificate":{"ACMCertificateArn":"${aws_acm_certificate.cert_1.arn}","MinimumProtocolVersion":"TLSv1.2_2021","SSLSupportMethod":"sni-only"},"Aliases":{"Quantity":1,"Items":["example.com"]},"Comment":"","Enabled":true}' --query 'Distribution.Id' --output text)
      echo "$ID2" >> .cloudfront_dist_ids

      # Dist 3: ACM Cert 2
      ID3=$(AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test AWS_DEFAULT_REGION=us-east-1 aws --endpoint-url=http://localhost:4566 cloudfront create-distribution --distribution-config '{"CallerReference":"dist_acm_2","Origins":{"Quantity":1,"Items":[{"Id":"myS3Origin","DomainName":"${aws_s3_bucket.test_bucket.bucket_regional_domain_name}"}]},"DefaultCacheBehavior":{"TargetOriginId":"myS3Origin","ViewerProtocolPolicy":"https-only"},"ViewerCertificate":{"ACMCertificateArn":"${aws_acm_certificate.cert_2.arn}","MinimumProtocolVersion":"TLSv1.1_2016","SSLSupportMethod":"sni-only"},"Aliases":{"Quantity":1,"Items":["test.com"]},"Comment":"","Enabled":true}' --query 'Distribution.Id' --output text)
      echo "$ID3" >> .cloudfront_dist_ids
    EOT
  }

  provisioner "local-exec" {
    when    = destroy
    command = <<-EOT
      set -eu
      if [ -f .cloudfront_dist_ids ]; then
        while read ID; do
          if [ -n "$ID" ]; then
            # We get the ETag needed for deletion
            ETAG=$(AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test AWS_DEFAULT_REGION=us-east-1 aws --endpoint-url=http://localhost:4566 cloudfront get-distribution-config --id "$ID" --query 'ETag' --output text || echo "")
            if [ -n "$ETAG" ] && [ "$ETAG" != "None" ]; then
              # Disable first (some LocalStack versions require this, just like real AWS)
              AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test AWS_DEFAULT_REGION=us-east-1 aws --endpoint-url=http://localhost:4566 cloudfront get-distribution-config --id "$ID" > config_$ID.json || true
              if [ -f config_$ID.json ]; then
                # Extract DistributionConfig and set Enabled=false
                jq '.DistributionConfig.Enabled = false | .DistributionConfig' config_$ID.json > updated_$ID.json
                NEW_ETAG=$(AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test AWS_DEFAULT_REGION=us-east-1 aws --endpoint-url=http://localhost:4566 cloudfront update-distribution --id "$ID" --if-match "$ETAG" --distribution-config file://updated_$ID.json --query 'ETag' --output text || echo "$ETAG")
                AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test AWS_DEFAULT_REGION=us-east-1 aws --endpoint-url=http://localhost:4566 cloudfront delete-distribution --id "$ID" --if-match "$NEW_ETAG" || true
                rm -f config_$ID.json updated_$ID.json
              else
                AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test AWS_DEFAULT_REGION=us-east-1 aws --endpoint-url=http://localhost:4566 cloudfront delete-distribution --id "$ID" --if-match "$ETAG" || true
              fi
            fi
          fi
        done < .cloudfront_dist_ids
        rm -f .cloudfront_dist_ids
      fi
    EOT
  }
}

