# _scratch

Throwaway probes used to establish how a provider's API actually behaves when
its documentation is incomplete or wrong.

The leading underscore is deliberate: the Go tool ignores such directories, so
nothing here is part of `go build ./...`, `go test ./...` or the shipped binary.
These are experiments, not code we maintain.

Run one with an explicit file path:

```sh
MIGADU_ACCOUNT=you@example.com MIGADU_API_KEY=... \
  go run ./_scratch/migadu-probe/main.go -domain example.com
```

## Why these exist

Migadu's Admin API is documented as early beta and the published surface is
incomplete. Several endpoints this project depends on are absent or hard to
find in the docs, and were established only by probing:

- `GET /domains/{d}/records` returns the DNS bundle including the per-domain
  `hosted-email-verify` token. Every intuitively named path (`/dns`,
  `/instructions`, `/verify`, `/zone`) returns 404.
- `PATCH /domains/{d}` with a cross-domain `catchall_destinations` value
  returns HTTP 200 but silently rewrites it to the same local part on the
  target domain. The 200 is not confirmation that the value was accepted.

That second finding is the reason these probes read state back after every
write instead of trusting a success status.
