# Security

## Reporting

Please report vulnerabilities privately via GitHub's **Report a vulnerability** (Security tab), not in public issues.

## Scope and known risks

- **Secrets.** Keys are read from the environment or a local `.env`, which is git-ignored. Use Binance **Demo Mode**
  keys only; the code has no production trading endpoint.
- **Prompt injection.** Headlines are untrusted text sent to a model. Typed outputs constrain the answer's format,
  not its truth. The gate thresholds, fixed notional, symbol allow-list and kill switch bound the impact; they do
  not eliminate it. Do not connect this to real funds.
- **Dependencies.** CI runs `govulncheck`; Dependabot tracks Go modules and Actions.
