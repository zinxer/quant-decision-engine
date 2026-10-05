# Contributing

Thanks for looking. The most useful contributions are **evidence**: measurements and hard cases that test the claims in the README.

## Most wanted

1. **Self-hosted latency numbers.** Run the benchmark next to your own Clef-flash server and open a PR adding a
   row to the results table in the README:
   ```bash
   CLEF_URL=http://localhost:8000/v1/systemone CLEF_MODEL=clef-flash \
     go run ./cmd/qde --bench 500 --bench-warmup 20 --label "1x H100, vLLM, same host"
   ```
   Include hardware, serving stack, and whether client and server share a host or VPC.
2. **Hard headlines.** Add cases to `data/headlines.json` that break either classifier, with the expected label
   and a one-line reason in the PR. Re-run `make record` and commit the updated recording.
3. **Prompt improvements** to the criteria in `internal/decision/clef.go`, with before/after `make record` output.

## Workflow

```bash
make test     # vet + race tests (must pass)
make lint     # golangci-lint
make cover    # coverage summary
```

- Keep `go run ./cmd/qde` working with no keys.
- Never commit `.env`, API keys, or hand-written "recordings".
- Significant design changes get an ADR in `docs/adr/`.
