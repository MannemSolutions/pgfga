# Developing pgfga

## History

Pgfga was initially developed in-house by [bol.com](https://www.bol.com) as a private python solution called pgcdfga.
The code was later renamed to pgfga, made publicly available,  and converted to GoLang by Mannem Solutions.
Mannem Solutions is committed to keep improving the project.

## Contributing

If you want to contribute, please do.
Just submit a Pull Request, and we will work from there.

## Versioning
pgfga uses semantic versioning, which means:
- breaking changes will always be a new major release
- new features will always be a new major or minor release
- hotfixes will always be introduced in a new major, minor or patch release.

# Make

## Make build

Will build the software.

```
make build
```

## Make debug

Will start dlv to debug the software.

```
make debug
```

### Make test

Runs gosec and golangci-lint.

```
make test
```

### Make inttest

Runs docker compose to setup 3 containers:
- an openldap container
- a postgres container
- a pgfga container that will sync users and groups from openldap into postgres and set up the other configured objects

```
make inttest
```

### Make run

Runs the built software, using variables in Dockerfile to determine image name, version and project. Using the `--rm` flag to keep the system clean.

```
make run
```

### Make all

Will run the inttest as a demo

```
make
```

