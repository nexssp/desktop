# Security policy

Please report suspected vulnerabilities privately to the repository maintainers rather than opening a public issue with exploit details. Include the affected module version, platform, reproduction steps, and impact. Do not include real user data, credentials, or private application files.

This library binds loopback listeners only to literal loopback IP addresses, uses owner-only defaults for persisted helper files, and does not collect telemetry. Applications remain responsible for their own authentication, authorization, HTTP origin checks, secret storage, and update process.
