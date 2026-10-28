# shorturl

A small Go library for creating, resolving, expiring, and revoking URL mappings.
It uses caller-owned storage and the Go standard library. It does not run an HTTP
server, fetch destinations, resolve DNS, or open storage connections.

The module path remains `shortened_url` for compatibility with this repository.
Import it with the package alias `shorturl`. Go 1.20 or newer is required.

## Installation and usage

The current module path is a local module name, not a publicly fetchable repository
path. To use this checkout from another module, run:

```sh
go mod edit -require=shortened_url@v0.0.0
go mod edit -replace=shortened_url=/absolute/path/to/shortened-url
go mod tidy
```

Run `go mod tidy` after adding the import. Within this checkout, run the working
example with `go run ./examples/basic`.

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    shorturl "shortened_url"
)

func main() {
    service, err := shorturl.New(&shorturl.MemoryStore{}, shorturl.Config{})
    if err != nil {
        log.Fatal(err)
    }
    ctx := context.Background()
    link, err := service.Shorten(ctx, "https://example.com/docs?a=1&b=2#start", time.Hour)
    if err != nil {
        log.Fatal(err)
    }
    destination, err := service.Resolve(ctx, link.Code)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(link.Code, destination) // Use only public example data in logs.
    if err := service.Revoke(ctx, link.Code); err != nil {
        log.Fatal(err)
    }
}
```

The library returns a code rather than a complete public short URL. Applications
choose their public origin and any routing. Each `Shorten` call creates a separate
mapping, even for the same destination. A zero TTL creates a permanent link;
negative TTLs are rejected. A positive TTL starts at creation and expires exactly
when `now >= ExpiresAt`, whether or not any cleanup runs. Revocation is idempotent
for an existing record and takes precedence over expiration during resolution.

## Defaults and validation

`New(store, Config{})` allows HTTPS only and limits destination input to 8192 bytes
before parsing. Set `AllowHTTP: true` to permit HTTP explicitly. `MaxURLBytes: 0`
selects the default; negative limits and nil stores are invalid. Configuration is
copied. A typed nil store is also rejected.

`service.ValidateURL(input)` uses the same policy as creation and resolution:

- Require an absolute hierarchical URL, a host, and HTTPS (or explicitly enabled HTTP).
- Reject userinfo, relative URLs, backslashes, raw controls, invalid UTF-8, leading
  or trailing whitespace, and malformed percent escapes in every component.
  Percent-encoded ASCII controls and backslashes are also rejected.
- DNS labels must be ASCII letters, digits, or interior hyphens, at most 63 bytes
  each and 253 bytes overall. Single-label names are allowed; trailing dots,
  underscores, empty labels, and Unicode hostnames are rejected. ASCII punycode
  labels are accepted syntactically, without IDNA conversion or visual trust checks.
- Canonical dotted-decimal IPv4 and bracketed IPv6 literals are accepted, including
  IPv4-mapped IPv6. IPv6 zones, shortened/octal/hex IPv4 spellings, and numeric or
  `0x`-prefixed final DNS labels are rejected to avoid ambiguous IP interpretations.
- Explicit ports must be decimal 1–65535 without leading zeroes. All such ports
  are allowed; an omitted port follows the URL scheme's default.

Validation preserves the destination string exactly, including `www.`, host case,
path escaping, query order, and fragments. It performs no normalization or DNS
lookup. Private IPs, localhost, and internal DNS names are allowed: syntax
validation does not establish trust or public reachability. There is no host
allowlist in this API. Applications needing a stricter destination policy must
apply it at both creation and resolution.

Codes use 16 bytes from `crypto/rand`, encoded as 22-character unpadded base64url.
`ValidateCode` requires canonical decoding and re-encoding of exactly 16 bytes.
Codes are case-sensitive. Only atomic insertion conflicts trigger retries, with
five total attempts before `ErrCodeExhausted`. Existing mappings are never overwritten.

## Storage integration

Implement this interface and pass an instance to `New`:

```go
type Store interface {
    Insert(context.Context, shorturl.Link) error
    Get(context.Context, string) (shorturl.Link, error)
    Revoke(context.Context, string) error
}
```

The consuming application owns storage construction, connections, and shutdown.
All implementations must:

- Support concurrent use, honor cancellation, and return independent value copies.
- Atomically insert without replacement, returning `ErrConflict` for reserved codes.
- Distinguish `ErrNotFound` from operational failures. Preserve wrapped sentinels
  using Go error wrapping, and avoid attaching multiple contradictory classifications.
- Atomically revoke an existing record, succeeding again if it is already revoked.
  Return `ErrNotFound` for a missing record. A resolve starting after successful
  revocation must see that revocation; an already in-flight resolve may complete.
  Lagging replicas or caches that break this guarantee do not satisfy the contract.
- Keep destinations and codes immutable. Never reuse a code for a different
  destination, even after expiration or revocation. Keep records or permanent
  code reservations if removing payloads under a retention policy.

`Link` contains `Code`, `Destination`, UTC `CreatedAt`, `ExpiresAt` (zero for no
expiration), and `Revoked`. Resolution verifies the returned code, checks
revocation and expiration, then revalidates the destination against current policy.
Policy changes can therefore disable previously accepted destinations.

`MemoryStore` is a zero-value-usable, race-safe reference implementation. It is
**not durable production storage**: it loses data on exit and grows without bound.
Service operations are safe for concurrent use with a conforming store. There
are no background cleanup jobs or caches. Store calls can succeed even when a
caller loses its response or cancellation races with completion; retrying creation
may produce another link. The API offers no idempotency keys.

## Errors

Use `errors.Is` to inspect returned errors:

| Error | Meaning |
| --- | --- |
| `ErrInvalidConfig` | Missing store or invalid configuration |
| `ErrInvalidURL`, `ErrInvalidCode`, `ErrInvalidTTL` | Invalid input or destination rejected on resolution |
| `ErrNotFound` | No record exists |
| `ErrExpired`, `ErrRevoked` | Record is disabled |
| `ErrConflict` | Store insertion encountered a reserved code; handled internally by shortening |
| `ErrCodeExhausted` | All five insertion attempts conflicted |
| `ErrRandom` | Cryptographic random bytes were unavailable |
| `ErrStorage` | Operational storage failure or mismatched record code |

Storage errors have generic text and retain `errors.Is(err, backendError)` checks
without exposing backend text or providing `Unwrap`/`errors.As` access to backend
objects. Context cancellation and deadlines remain inspectable with `errors.Is`.
The library checks context before storage access and after calls return; an adapter
must cooperate to interrupt an ongoing operation. It does not log URLs or errors.

## Security and publication boundaries

Short codes are not authentication or authorization. Anyone possessing a public
link can resolve it. Applications own authentication, authorization, rate limiting,
TLS, redirect handling, abuse response, and sensitive-data retention. Do not log
full destination URLs or query strings. URL validation does not prevent phishing
or establish SSRF protection for applications that fetch destinations.

This code has not undergone an independent security audit. No license was added:
the owner must choose a license and confirm publication rights. A public module
path also needs an explicit decision before publication.

## Legacy compatibility

The previous packages `shortened`, `entity`, and `consts`, and root demonstration
server have been replaced. Import `shortened_url` as `shorturl`, create a service,
and use `Shorten`/`Resolve`/`Revoke` with contexts. `Extend`, `TransformWWW`, and
`ValidLink` are removed; `Link` replaces the old `entity.URL` shape.

Before replacement, characterization tests passed for six-letter codes, repeated
URL deduplication, rewriting bare hosts/HTTP/`www.` to HTTPS without `www.`, missing
code errors, unsupported-scheme rejection, and permissive acceptance of empty or
malformed inputs by `ValidLink`. These unsafe behaviors are deliberately changed:
new inputs must be absolute URLs, HTTP is opt-in, destinations retain exact spelling,
new codes are cryptographic, and destination deduplication is removed. Permanent
links, query order, and fragments remain supported. The old in-memory maps were
not persistent and contained no export or migration mechanism.

Existing six-letter codes fail the new canonical validator. Applications with
externally preserved legacy mappings should keep a separate read-only legacy
resolver with a bounded six-ASCII-letter grammar and validate its destinations.
Route that distinct grammar separately from new 22-character codes; do not relax
`ValidateCode`. Preserve original stored destinations and code reservations. Never
fall back after revocation, expiration, policy rejection, or storage failure.
Data migration and cutover belong to the application; this library does not migrate
or silently rewrite legacy mappings.

## Verification

```sh
go test ./...
go test -race ./...
go vet ./...
go test -run='^$' -fuzz=FuzzValidateURL -fuzztime=10s
go test -run='^$' -fuzz=FuzzValidateCode -fuzztime=10s
go run ./examples/basic
# When installed:
govulncheck ./...
```
