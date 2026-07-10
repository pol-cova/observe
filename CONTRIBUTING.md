# Contributing to observe

Thanks for helping improve `observe`.

## Before opening a pull request

1. Fork the repository and create a focused branch.
2. Keep changes small and explain the user-facing outcome in the pull request.
3. Format and verify the code locally:

   ```bash
   gofmt -w .
   go vet ./...
   go test ./...
   go build ./...
   ```

4. Update the README when a command, flag, or supported integration changes.

## Reporting bugs

Include your operating system, `observe` version, the command you ran, expected behavior, and actual behavior. Remove secrets, tokens, and private hostnames from logs before sharing them.

## Code guidelines

- Prefer standard-library Go and small, readable functions.
- Keep the default experience fast and configuration-free.
- Do not add telemetry or send data from the machine without explicit user consent.

By participating, you agree to follow the [Code of Conduct](CODE_OF_CONDUCT.md).
