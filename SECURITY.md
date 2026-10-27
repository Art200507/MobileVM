# Security status

ShadowDevice is an unaudited development prototype. Use synthetic data on trusted
machines. The implemented boundary protects disk contents against a ciphertext
store that lacks keys; it does not protect against compromise of the local client.

Reader trust anchors and exact snapshot digests must be provisioned independently
of storage. Signed versions alone cannot identify the latest state on a fresh
recovery device. The local writer trusts filesystem durability and head
serialization; CAS does not prevent a malicious authority from forking views.

Identity files contain plaintext key material. Keep them outside served directories
and use the reader file rather than the owner signing file for reads. Local
plaintext output, the optional loopback range endpoint, process memory, crash dumps,
swap, and runtime key copies all lie inside the trusted development-host boundary.
No endpoint-erasure guarantee is made.

The development servers bind to numeric loopback addresses. They do not provide
production authentication, authorization, quotas, tenancy, TLS provisioning or
rate limits. Do not expose them through proxies/tunnels to untrusted users. Storage
metadata reveals disk geometry, object references, reuse and access timing.

Recovery, sessions and leases remain unavailable. Do not substitute an ordinary
server-side PIN counter for a compromise-resistant recovery protocol. No secret
is released on a simulated attestation result. Future Android recovery must
measure the payload, authenticate the delivery endpoint and define human
input/output trust; protected memory alone is insufficient.

A high-entropy recovery kit, device migration, key/signer rotation, protected
execution, service outages, garbage collection, write-back durability, browser
mode and lender consent still need separate implementations and verification.
See docs/remaining-work.md for the current gates.
