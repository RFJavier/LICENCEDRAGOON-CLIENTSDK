# Changelog

All notable changes to this project will be documented in this file.

## [0.9.0] - 2026-04-01
### Added
- Ed25519 signature verification in SDK responses.
- Nonce support in SDK HTTP client requests.
- Heartbeat loop with grace period hooks.
- File-based state storage for offline tolerance.
- Unit tests for SDK package.

### Changed
- Public endpoints aligned to /v1/license/activate and /v1/license/heartbeat.
- Config naming uses APIURL and PublicKey.

### Notes
- This is the initial public beta release of the separated Go SDK repository.
