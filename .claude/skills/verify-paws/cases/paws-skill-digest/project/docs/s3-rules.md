# S3 rules
1. Block all public access: BlockPublicAcls, IgnorePublicAcls, BlockPublicPolicy, RestrictPublicBuckets.
2. Encrypt with SSE-S3 (AES256) or SSE-KMS.
3. Tag every bucket with owner, team and created-by=ai-agent. Rule id S3-TAG-7731.
