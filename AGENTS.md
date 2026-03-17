# Repository Guidelines

## Project Structure & Module Organization
This Mattermost plugin has two main parts:

- `server/`: Go backend for plugin lifecycle, API handlers, slash commands, and configuration.
- `webapp/src/`: React frontend for UI components, Redux state, and plugin entry points.
- `build/`: manifest tooling and shared build helpers.
- `plugin.json`: plugin metadata; keep it in sync with generated manifest code.
- `webapp/src/components/`: feature components with colocated tests and snapshots.

Example paths: `server/plugin.go`, `server/command.go`, `webapp/src/plugin.jsx`, `webapp/src/components/translated_message/`.

## Build, Test, and Development Commands
- `make apply`: propagates manifest metadata into server and webapp code.
- `make check-style`: runs `gofmt`, `go vet`, and webapp ESLint checks.
- `make test`: runs Go tests in `server/...` and frontend tests in `webapp/`.
- `make dist`: builds server binaries, bundles the webapp, and creates the plugin archive in `dist/`.
- `make clean`: removes build outputs, coverage files, and installed webapp dependencies.
- `cd webapp && npm run test:watch`: runs Jest in watch mode during UI work.

Run commands from the repository root unless a command explicitly targets `webapp/`.

## Coding Style & Naming Conventions
Use Go defaults: tabs for indentation, `gofmt` formatting, and idiomatic mixedCaps names. Keep server files focused by responsibility, following existing patterns like `api.go`, `command.go`, and `configuration.go`.

In the webapp, use ES6 modules, 4-space indentation only if already present in the file, and prefer existing naming patterns: React components in `*.jsx`, helpers in `*.js`, directories in lowercase with underscores only when already established. Use `npm run lint` or `make check-style` before opening a PR.

## Testing Guidelines
Backend tests run with `go test -v -race ./server/...`. Frontend tests use Jest and React Testing Library via `cd webapp && npm test`.

Name Go tests with standard `_test.go` files. Name frontend tests `*.test.js` and keep snapshots next to the tested component under `__snapshots__/`. Add or update tests for behavior changes, especially around commands, API responses, reducers, and rendered post UI.

## Commit & Pull Request Guidelines
Recent history favors concise, imperative subjects, often with prefixes like `feat:`, `fix:`, or bracketed scopes such as `[chore]`. Keep commits focused and easy to review.

Pull requests should include a short summary, linked issue or ticket when available, test notes, and screenshots or GIFs for visible webapp changes. Call out config or manifest changes explicitly so reviewers can verify plugin packaging and deployment behavior.
