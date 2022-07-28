# Vroom

Vroom is the profiling service returning profiles and aggregated data about your profiles. It's written in Go.

The service is currently hosted on Google Cloud Run and requires access to Google Cloud Storage.

## Development

In order to develop for `vroom`, you will need:
- `golang` >= 1.18
- `make`
- Docker and Docker Compose
- Snuba (via Sentry development services)
- `pre-commit`

In order to install `pre-commit`, you will need `python` and run:
```sh
pip install --user pre-commit
```

### Build development server

```sh
make dev
```

### Run tests

```sh
make test
```

## Deploy

A deploy will be automatic when your PR gets merged to `main`. Otherwise, you could build the Docker image and deploy manually with:
```sh
make docker deploy
```
