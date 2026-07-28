# Security Policy

## Supported versions

Security fixes are provided for the latest published release.

## Reporting a vulnerability

Do not disclose suspected vulnerabilities in a public issue.

Use GitHub's private vulnerability reporting feature in the **Security** tab of
this repository. Include the affected version, reproduction steps, expected
impact and any suggested mitigation.

Maintainers should acknowledge a report within five business days. Publication
of details should wait until a fix or mitigation is available.

Never include Docker credentials, TLS private keys, certificates, tokens,
environment files or production configuration in a report.

## Dependency scanning

CodeQL, Dependabot and `govulncheck` run automatically. The Docker SDK shares
its module with Moby daemon code, so `govulncheck` may report daemon-only
advisories even though DockTop is a client and does not expose affected
operations such as `docker cp` or plugin management. These findings remain
visible and must be reviewed when Docker publishes a new SDK version.
