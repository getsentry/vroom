# vroom

`vroom` is the profiling service processing profiles and deriving data about your profiles. It's written in Go.

The name was inspired by this [video](https://www.youtube.com/watch?v=t_rzYnXEQlE).

## Development

In order to develop for `vroom`, you will need:
- `golang` >= 1.18
- `make`
- Snuba (via Sentry development services)
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
