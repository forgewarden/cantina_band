# Security Policy

## Supported Versions

Only the latest stable release receives security updates. Deployments should use an immutable image digest and regularly update to the digest published for the latest stable release.

## Reporting a Vulnerability

Do not report suspected vulnerabilities in a public issue. Use [GitHub private vulnerability reporting](https://github.com/forgewarden/cantina_band/security/advisories/new) and include:

- The affected version or image digest
- Steps to reproduce the issue
- The expected and observed behavior
- The potential impact
- Any known mitigations

You should receive an acknowledgement within seven days. Confirmed vulnerabilities will be assessed, fixed on a private branch, and disclosed with remediation guidance after a patched release is available.

## Release Artifacts

Official container images are published at `ghcr.io/forgewarden/cantina_band`. Stable images are signed using GitHub Actions workload identity and include build-provenance and SBOM attestations.

Security exceptions must identify the relevant vulnerability, explain the impact assessment, name an owner, and include an expiry date.
