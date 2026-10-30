# Authentication

Pocket Web implements Google and GitHub OAuth **architecture** using authorization-code flow with PKCE. Provider credentials are not configured on this host.

## Start

`POST /api/oauth/{google|github}/start` generates:

- a cryptographically random verifier (32 random bytes, base64url encoded);
- an S256 challenge computed as base64url(SHA-256(verifier));
- a high-entropy `state` value associated with the provider and a 10-minute expiry.

The response is labeled `"mode":"mock"` because no provider authorization redirect or credentials are configured.

## Callback validation

`POST /api/oauth/{provider}/callback` requires `state` and `codeVerifier`. The handler:

- looks up and deletes the state (single use);
- rejects unknown or expired state;
- rejects a verifier that doesn't match the started flow;
- then reports provider credentials as unavailable rather than fabricating a login.

The integration test independently recomputes the SHA-256 challenge, verifies a mismatched verifier is rejected, and verifies the same state cannot be replayed.

## Mock authentication

`POST /api/auth/mock` creates a clearly labeled `mode:"mock"` account. This is not a Google/GitHub login and the UI labels live provider authentication as unconfigured. Mock account data supports a complete offline demonstration.

## Credential storage

`CredentialStore` is an interface implemented by `AESGCMCredentialStore`:

- 32-byte key loaded/created under the Pocket Web data directory in `credential.key` with mode `0600`;
- AES-256-GCM with a random nonce per token;
- only ciphertext is stored in `oauth_tokens`;
- API responses never return token material.

A test supplies a token, verifies the stored bytes differ from plaintext and decrypt successfully, and checks that `/api/accounts` and the mock-auth response do not expose it. No plaintext token or cookie is logged or committed.

## Required deployment configuration

To perform real provider login, configure provider client IDs/redirect URIs and a server-side client secret/token exchange implementation. The present code intentionally returns provider-unavailable rather than claiming success. Do not commit credentials.
