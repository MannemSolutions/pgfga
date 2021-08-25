# Download and run

## Direct download
[pgfga](https://github.com/MannemSolutions/pgfga) is available for download for many platforms and architectures from the [Github Releases page](https://github.com/MannemSolutions/pgfga/releases).
It could be as simple as:
```bash
PGFGA_VERSION=v2.0.0
cd $(mktemp -d)
curl -Lo "pgfga-${PGFGA_VERSION}-linux-amd64.tar.gz" "https://github.com/MannemSolutions/pgfga/releases/download/${PGFGA_VERSION}/pgfga-${PGFGA_VERSION}-linux-amd64.tar.gz"
tar -xvf "./pgfga-${PGFGA_VERSION}-linux-amd64.tar.gz"
mv pgfga /usr/local/bin
cd -
```

After that you can run pgfga directly from the prompt:
```bash
pgfga -c ./myconfig.yml
```

## Container image
For container environments [pgfga](https://github.com/MannemSolutions/pgfga) is also available on [dockerhub](https://hub.docker.com/repository/docker/mannemsolutions/pgfga).
You can easilly pull it with:
```bash
docker pull mannemsolutions/pgfga
```

The image has an example config.yaml, but you probably want to mount your own config file at /etc/pgfga/config.yaml:
```bash
docker run -v $PWD/config.yaml:/etc/pgfga/config.yaml mannemsolutions/pgfga
```
**Note** that the $PWD is added to mount the file with its full absolute path. Relative paths are not supported.

## docker-compose
You can use pgfga with docker compose.
The docker-compose.yml file could have contents like this:
```yaml
services:
  pgfga:
    image: mannemsolutions/pgfga
    volume:
      - ./testdata/config.yaml:/etc/pgfga/config.yaml
  postgres:
    image: postgres:13
    environment:
      POSTGRES_HOST_AUTH_METHOD: 'md5'
      POSTGRES_PASSWORD: pgfga
  ldap:
    image: osixia/openldap
    command:
      - "--copy-service"
      - "--loglevel"
      - debug
    environment:
      LDAP_ORGANISATION: pgfga
      LDAP_DOMAIN: pgfga.org
      LDAP_ADMIN_PASSWORD: pGfGa
    volumes:
      - ./testdata/ldif:/container/service/slapd/assets/config/bootstrap/ldif/custom
```
**Note** that the ldap needs content.
Please see the [github project for pgfga](for a working example) of docker-compose setting up an ldap, and postgres, and running pgfga against it, which consists of.
- The [docker-compose.yml file](https://github.com/MannemSolutions/pgfga/blob/docs/docker-compose.yml)
- The [bash script running docker compose](https://github.com/MannemSolutions/pgfga/blob/docs/docker-compose-tests.sh)
- The [ldif file we use](https://github.com/MannemSolutions/pgfga/blob/docs/testdata/ldif/01_objects.ldif)
- The [pgfga config file we use](https://github.com/MannemSolutions/pgfga/blob/docs/testdata/config.yaml)

## Direct build

Although not advised, you can also directly build from source:
```bash
go install github.com/mannemsolutions/pgfga/cmd/pgfga@master
```

After that you can run pgfga directly from the prompt:
```bash
pgfga -c pgfgaconfig.yml
```
