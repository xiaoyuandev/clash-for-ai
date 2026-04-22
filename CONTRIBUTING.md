# Contributing to Clash for AI

Thanks for contributing.

This project is still evolving, so the goal of this guide is to keep contributions easy to review and safe to merge.

## Before You Start

Please keep these principles in mind:

1. Prefer small, focused pull requests.
2. Avoid unrelated refactors in the same change.
3. Preserve the existing product direction unless the change explicitly proposes a shift.
4. Do not commit secrets, API keys, local runtime data, or internal-only documents.

## Development Setup

Requirements:

1. Node.js
2. pnpm
3. Go toolchain, if you need to build or run the core service locally

Install dependencies:

```bash
pnpm install
```

Run the desktop app in development mode:

```bash
pnpm dev
```

Type-check the desktop app:

```bash
pnpm --filter desktop typecheck
```

Build the desktop app:

```bash
pnpm --filter desktop build
```

## Repository Structure

```text
apps/desktop   Electron desktop application
core/          Go local gateway and provider management backend
docs/          Public documentation
```

## Contribution Scope

Good contribution types:

1. Bug fixes
2. UI and UX improvements
3. Logging and observability improvements
4. Better error handling
5. Documentation improvements
6. Packaging and release workflow improvements

If you want to make a large architecture change, open an issue or discussion first before sending a large PR.

## Branches and Pull Requests

Please follow this workflow:

1. Create a feature branch from `main`
2. Make focused changes
3. Run relevant validation locally
4. Open a pull request back to `main`

`main` is a protected branch.

Current repository rules:

1. Do not push directly to `main`
2. Do not force-push to `main`
3. Open a pull request for every change
4. Wait for the required `ci` check to pass
5. Wait for at least one approval before merge
6. Code owner review is required
7. Resolve review conversations before merge

If new commits are pushed to a PR, previous approvals may need to be re-confirmed.

Recommended PR content:

1. What changed
2. Why it changed
3. How it was tested
4. Any known limitations or follow-up work

## Validation

Before opening a PR, run what is relevant for your change.

At minimum for desktop-facing changes:

```bash
pnpm --filter desktop typecheck
pnpm build
```

If your change affects runtime behavior, also test it manually.

Examples:

1. Provider creation
2. Provider activation
3. Request logging
4. Desktop boot flow
5. Settings page behavior

## Coding Expectations

1. Keep changes pragmatic and easy to review.
2. Prefer simple implementations over clever abstractions.
3. Match the existing code style and structure.
4. Do not introduce unnecessary dependencies.
5. Add comments only where they genuinely improve readability.

## Documentation Rules

Public repository documentation should remain public-facing.

Do not add:

1. Internal product plans
2. Private roadmap notes
3. Internal release procedures that are not meant to be public
4. Secrets or internal infrastructure details

## Sensitive Data

Never commit:

1. Provider API keys
2. Tokens
3. Passwords
4. Local runtime data under ignored directories
5. Internal-only planning documents

Before pushing, search your changes for obvious secrets.

## Issues

When reporting a bug, include:

1. What you expected
2. What actually happened
3. Steps to reproduce
4. Logs or screenshots if relevant

## License

By contributing to this repository, you agree that your contributions will be licensed under the repository license.
