# Cleanup

Simple script to cleanup profiles that's stored on local filesystem daily, used by self-hosted repository.

It requires 2 environment variables:
- `SENTRY_BUCKET_PROFILES` - path to profiles directory
- `SENTRY_EVENT_RETENTION_DAYS` - retention days for profiles, in plain numbers (sample: 90, not 90d). A common environment variable on self-hosted (also used by Sentry and Snuba service)