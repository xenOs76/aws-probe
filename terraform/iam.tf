locals {
  iam_tags = {
    Project = "aws-probe"
  }
}

resource "aws_iam_user" "aws_probe_assumer" {
  name = "aws-probe-assumer"
  tags = local.iam_tags
}

resource "aws_iam_access_key" "aws_probe_assumer" {
  user = aws_iam_user.aws_probe_assumer.name
}

resource "aws_iam_policy" "s3_read_only" {
  name = "aws-probe-s3-read-only"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = "s3:ListAllMyBuckets"
        Resource = "*"
      },
      {
        Effect   = "Allow"
        Action   = "s3:ListBucket"
        Resource = aws_s3_bucket.test_bucket.arn
      },
    ]
  })
  tags = local.iam_tags
}

resource "aws_iam_policy" "s3_read_write" {
  name = "aws-probe-s3-read-write"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = "s3:ListBucket"
        Resource = aws_s3_bucket.test_bucket.arn
      },
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject",
          "s3:DeleteObject",
        ]
        Resource = "${aws_s3_bucket.test_bucket.arn}/*"
      },
    ]
  })
  tags = local.iam_tags
}

resource "aws_iam_policy" "secret_1_read_only" {
  name = "aws-probe-secret-1-read-only"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue",
          "secretsmanager:DescribeSecret",
        ]
        Resource = aws_secretsmanager_secret.test_secret_1.arn
      },
    ]
  })
  tags = local.iam_tags
}

resource "aws_iam_policy" "secret_2_read_only" {
  name = "aws-probe-secret-2-read-only"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue",
          "secretsmanager:DescribeSecret",
        ]
        Resource = aws_secretsmanager_secret.test_secret_2.arn
      },
    ]
  })
  tags = local.iam_tags
}

resource "aws_iam_role" "s3_read_only" {
  name = "aws-probe-s3-read-only"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect    = "Allow"
        Principal = { AWS = aws_iam_user.aws_probe_assumer.arn }
        Action    = "sts:AssumeRole"
      },
    ]
  })
  tags = local.iam_tags
}

resource "aws_iam_role" "s3_read_write" {
  name = "aws-probe-s3-read-write"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect    = "Allow"
        Principal = { AWS = aws_iam_user.aws_probe_assumer.arn }
        Action    = "sts:AssumeRole"
      },
    ]
  })
  tags = local.iam_tags
}

resource "aws_iam_role" "secret_1_read_only" {
  name = "aws-probe-secret-1-read-only"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect    = "Allow"
        Principal = { AWS = aws_iam_user.aws_probe_assumer.arn }
        Action    = "sts:AssumeRole"
      },
    ]
  })
  tags = local.iam_tags
}

resource "aws_iam_role" "secret_2_read_only" {
  name = "aws-probe-secret-2-read-only"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect    = "Allow"
        Principal = { AWS = aws_iam_user.aws_probe_assumer.arn }
        Action    = "sts:AssumeRole"
      },
    ]
  })
  tags = local.iam_tags
}

resource "aws_iam_role_policy_attachment" "s3_read_only" {
  role       = aws_iam_role.s3_read_only.name
  policy_arn = aws_iam_policy.s3_read_only.arn
}

resource "aws_iam_role_policy_attachment" "s3_read_write" {
  role       = aws_iam_role.s3_read_write.name
  policy_arn = aws_iam_policy.s3_read_write.arn
}

resource "aws_iam_role_policy_attachment" "secret_1_read_only" {
  role       = aws_iam_role.secret_1_read_only.name
  policy_arn = aws_iam_policy.secret_1_read_only.arn
}

resource "aws_iam_role_policy_attachment" "secret_2_read_only" {
  role       = aws_iam_role.secret_2_read_only.name
  policy_arn = aws_iam_policy.secret_2_read_only.arn
}

resource "aws_iam_policy" "assume_roles" {
  name = "aws-probe-assume-roles"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = "sts:AssumeRole"
        Resource = [
          aws_iam_role.s3_read_only.arn,
          aws_iam_role.s3_read_write.arn,
          aws_iam_role.secret_1_read_only.arn,
          aws_iam_role.secret_2_read_only.arn,
        ]
      },
    ]
  })
  tags = local.iam_tags
}

resource "aws_iam_user_policy_attachment" "assume_roles" {
  user       = aws_iam_user.aws_probe_assumer.name
  policy_arn = aws_iam_policy.assume_roles.arn
}
