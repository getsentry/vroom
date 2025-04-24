<p align="center">
  <a href="https://sentry.io/?utm_source=github&utm_medium=logo" target="_blank">
    <picture>
      <source srcset="https://sentry-brand.storage.googleapis.com/sentry-logo-white.png" media="(prefers-color-scheme: dark)" />
      <source srcset="https://sentry-brand.storage.googleapis.com/sentry-logo-black.png" media="(prefers-color-scheme: light), (prefers-color-scheme: no-preference)" />
      <img src="https://sentry-brand.storage.googleapis.com/sentry-logo-black.png" alt="Sentry" width="280">
    </picture>
  </a>
</p>

# Sentry vroom

[![GitHub Release](https://img.shields.io/github/release/getsentry/vroom.svg)](https://github.com/getsentry/vroom/releases/latest)

<p align="center">
    <img src="https://github.com/getsentry/vroom/blob/main/artwork/vroom-logo.png?raw=true" alt="vroom" width="640">
</p>

`vroom` is Sentry's profiling service, processing and deriving data about your profiles. It's written in Go.

The name was inspired by this [video](https://www.youtube.com/watch?v=t_rzYnXEQlE).

## Development

In order to develop for `vroom`, you will need:
- `golang` >= 1.18
- `make`
- `pre-commit`

### pre-commit

In order to install `pre-commit`, you will need `python` and run:
```sh
pip install --user pre-commit
```

Once `pre-commit` is installed, you'll have to set up the actual git hook scripts with:
```sh
pre-commit install
```

### Build development server

```sh
make dev
```

### Run tests

```sh
make test
```

## Release Management

We use GitHub actions to release new versions. `vroom` is automatically released using Calendar Versioning on a monthly basis together with sentry (see https://develop.sentry.dev/self-hosted/releases/), so there should be no reason to create a release manually. That said, manual releases are possible with the "Release" action.
