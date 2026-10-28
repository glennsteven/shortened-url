# URL shortening library design

The implementation is a single root package, `shorturl`, in the public module
`github.com/glennsteven/shortened-url`. It requires Go 1.26 or newer and uses only the standard library.
The executable under `examples/basic` demonstrates creation, resolution, and revocation without a server.

## API and ownership

`New(Store, Config)` constructs a concrete `Service`. Callers own storage and its
lifecycle. `Shorten(ctx, destination, ttl)` returns a `Link`; `Resolve(ctx, code)`
returns the exact destination; `Revoke(ctx, code)` disables an existing link.
`ValidateURL` is a service method; `ValidateCode` is a package function. Configuration
contains only `AllowHTTP` and `MaxURLBytes`. No namespaces, destination updates,
custom aliases, deduplication, database adapters, HTTP handlers, caches, or jobs.

`Link` contains a code, destination, creation time, expiration time, and revocation
flag. TTL zero means permanent, preserving the legacy lifecycle default. Negative
TTLs fail. Positive TTLs expire at `now >= ExpiresAt`. Revocation takes precedence.
An internal clock allows precise boundary testing without exposing test machinery.

## Creation and resolution

Creation checks context, validates the destination and TTL, computes UTC times,
generates 16 cryptographic random bytes, and atomically inserts a new record.
Codes are 22-character unpadded base64url. Only `ErrConflict` retries, with five
total attempts. There is no existence check or overwrite. Randomness failures stop
immediately. Internal randomness injection exists only for deterministic tests.

Resolution checks context and canonical code encoding, then reads authoritative
storage. It verifies the returned code, rejects revoked or expired records, and
revalidates the stored destination under the service's current policy. It returns
the original string without normalization. There is no resolve cache.

## Destination policy

Default policy allows HTTPS and at most 8192 bytes before parsing. HTTP is opt-in.
Validation uses `net/url` plus explicit checks: hierarchical absolute URLs,
nonempty authority, no userinfo, no controls/backslashes or outer whitespace,
valid UTF-8 and percent escapes throughout the input. Escaped ASCII controls and
backslashes are rejected. Query strings and fragments retain exact spelling.

ASCII DNS labels are bounded to 63 bytes and total hosts to 253. Single-label hosts
and ASCII punycode are allowed; Unicode hostnames, trailing dots, underscores, and
empty labels are rejected. No IDNA conversion is performed. Canonical IPv4 and
bracketed IPv6 (including mapped IPv4) are supported; IPv6 zones and ambiguous
numeric/octal/hex IPv4 forms are rejected. Numeric and `0x`-prefixed final DNS labels
are conservatively rejected. Explicit ports must be canonical decimal 1–65535.

Private IPs and internal names are allowed. There are no network requests, DNS
lookups, public-address promises, or host allowlists. Applications needing stricter
policy must enforce it before creation and before using a resolved destination.
URL syntax does not establish destination trust, browser safety, or fetch safety.

## Storage and errors

`Store` has `Insert`, `Get`, and `Revoke`, all accepting contexts. It must be safe
for concurrent use. Inserts atomically reject duplicates with `ErrConflict`;
missing records return `ErrNotFound`; operational failures remain distinct.
Revocation is idempotent for existing records and authoritative for resolves
starting after successful revocation. Already in-flight resolves may complete.
Adapters must preserve immutable destinations and permanent code reservations,
including expired/revoked records. Cleanup must never permit code reassignment.

The included zero-value `MemoryStore` uses locks and returns value copies. It is
not durable, retains all reservations, and grows without bound. It illustrates
the contract; it is not production persistence. No schema or migration is supplied.

Public sentinels distinguish configuration, URL/code/TTL input, missing records,
expiration, revocation, conflict exhaustion, randomness, and storage failures.
Operational errors are wrapped with safe text and `errors.Is` classification;
backend messages and objects are not exposed through unwrapping. Context checks
before and after storage calls preserve cancellation; adapters must cooperate
to interrupt calls. Cancellation or response loss after a successful insert can
leave a mapping created, and retries can create another mapping.

## Legacy decisions and verification

The legacy implementation was inspected and characterized before replacement.
It used global, unsynchronized maps; MD5 destination deduplication; six-letter
`math/rand` codes; permissive regex validation; forced HTTPS/`www.` normalization;
and a permanent-redirect demonstration server. Those behaviors were not a safe
reusable API. The new service removes globals and normalization, rejects bare
hosts, defaults to HTTPS, creates independent mappings, and requires canonical
codes. Permanent links, exact query order, and fragments remain available.

Existing six-letter codes require a separate application-owned legacy resolver.
Do not weaken the new parser, rewrite stored destinations, or revive disabled
records through fallback. See README for the compatibility strategy. No automatic
migration is included.

Tests cover destination preservation and rejection, canonical codes, deterministic
collisions and exhaustion, storage failures and cancellation, exact expiration,
idempotent revocation, concurrent insert uniqueness and post-revoke reads, and
independent copies. URL and code fuzz targets exercise parser boundaries. Run
formatting, unit/race tests, vet, the example, fuzz smoke tests, and govulncheck
when available.

Consumers own authentication, authorization, TLS, abuse handling, rate limiting,
HTTP redirects, and storage operations. This library is not independently audited.
Publication still requires the owner's license/rights decision and verification of
GitHub review/CI enforcement. The intended first version is v0.1.0, not yet
published or verified for public installation. No license, release, commit, or push
is part of this preparation.

## Repository automation

CI and the tag-triggered release workflow share the verification in
`.github/workflows/ci.yml`. Only the release publishing job has contents write
permission. It publishes generated notes after tag/module-major validation,
main-branch ancestry verification, and successful checks against the tagged commit.
No deployment, storage infrastructure, or application artifacts are involved.
See [CONTRIBUTING.md](../CONTRIBUTING.md) for branch protection and the manual
release procedure. The confirmed public module path is compatible with v0.1.0.
A separate consumer build using a local replace directive verifies integration;
public availability requires retrieving and building an actually published tag
without a replace directive.
