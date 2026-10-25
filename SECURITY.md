# Security

Please do not post credentials, private tenant evidence or exploitable security
details in public issues. Use GitHub's private vulnerability reporting on the
repository Security tab. If private reporting is unavailable, open an issue asking
for a private reporting channel without disclosing the vulnerability.

The hosted sandbox accepts only fixed synthetic scenarios. It does not provide
tenant access, persist visitor evaluations, accept phone numbers, or run benchmark
jobs. Shared limits are per API process: 30 requests per minute and two concurrent
evaluations. This is a portfolio service, not a high-availability or production
verification service. Render can sleep; requests may need retrying after wake-up.

Run a private local instance for unrestricted exploration. Default local identities
and passwords are synthetic development fixtures, never hosted credentials.
