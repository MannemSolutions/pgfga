# Developing pgfga

## History

Pgfga was initially developed in-house by [bol.com](https://www.bol.com) as a private python solution called pgcdfga.
The code was later renamed to pgfga, made publicly available,  and converted to GoLang by Mannem Solutions.
Mannem Solutions is commited to keep improving the project.

## Contributing

If you want to contibute, please do. Just submit a Pull Request and we will work from there.

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

## Make build-image

Will build and push a docker container with pgfga.

```
make build-image
```

### Make test

Runs all available go tests like gosec, go test (unittest), golint, etc.
Note that all of these tests are also part of the github ci workflows.

```
make test
```

### Make inttest

Runs docker compose to setup 3 containers:
- an openldap container
- a postgres container
- a pgfga container that will sync users and groups from openldap into postgres and setup the other configured objects

```
make inttest
```

### Make run

Runs the built software, using variables in Dockerfile to determine image name, version and project. Using the `--rm` flag to keep the system clean.

```
make run
```

### Make all

Will build the container image locally and run the unittest with docker-compose as a demo.

```
make
```

