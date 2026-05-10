# Contributing

Thanks for your interest in improving Sweeper!

## How to contribute

- **Report bugs** — Open an issue with the output of `sweeper --version` and steps to reproduce.
- **Suggest features** — Open an issue describing the use case and desired behavior.
- **Submit pull requests** — Fork the repo, make your changes, and open a PR against `main`.

## Fingerprint contributions

The most impactful way to contribute is adding new fingerprints to `internal/matcher/fingerprints.go`. If Sweeper doesn't recognize an app's leftover folders, open an issue with the app name, bundle ID, and leftover folder paths found under `~/Library`.

## Development setup

```bash
git clone https://github.com/danorul9/sweeper.git
cd sweeper
make build
make test
```

Run `go vet ./...` before submitting a PR. All tests must pass.

## Guidelines

- Keep the single-binary, zero-dependency philosophy.
- Preserve the existing signal/confidence scoring model.
- Add tests for new matching strategies or fingerprints.
- Use `--json` and `--dry-run` flags for new commands.
