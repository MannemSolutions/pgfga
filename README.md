# PGCDFGA - Postgres Container Deployment Fine Grained Access tool

Tool to configure Postgres logical objects, being Users, Roles, Databases and Extensions.

The provided Makefile takes care of most docker/gcloud stuff.

## License

   Copyright 2019 Bol.com

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.

## Requirements

### PGCDFGA config

PGCDFGA can be configured using a configuration yaml.
In container deployments (like Kubernetes), that might be configmap mounted as a volume.

### Postgres User account

The PGCDFGA tool requires a postgres user with access and SUPERUSER privilleges to run.
The user could be made available either with:
- Local access (when running inside of master container) and using ident (or trust) authentication
- A username / password configured with setup.sql script (and setting PG_PRIMARY_USER, PG_PRIMARY_PASSWORD, etc.).
- A client certificate (by far the most secure solution)

The most convenient way is to use a client certificate, handed out to PGCDFGA user as configured in config.yaml (postgres.dsn).

## Make

## Make build

Will build the docker image from `Dockerfile`, using variables in `Dockerfile` to determine image name, version and project.

```
make build
```

### Make tag

Will tag images for usage with `gcloud`, using variables in `Dockerfile` to determine image name, version and project.

Will tag both latest and version.

```
make tag
```

### Make push

Will push latest and versioned images to gcloud, using variables in `Dockerfile` to determine image name and version.

```
make push
```

By default it will try to push to dockerhub.com/bol.com/pgcdfga. To override that, set the environmentvariable to the other projectname.

For example:
```
export PROJECTS=eu.gcr.io/my-cool-app-tst-proj eu.gcr.io/my-cool-app-pro-proj
make push
```
will push to the following projects:
* eu.gcr.io/my-cool-app-tst-proj/pgcdfga
* eu.gcr.io/my-cool-app-pro-proj/pgcdfga

### Make run

Runs the built image, using variables in Dockerfile to determine image name, version and project. Using the `--rm` flag to keep the system clean.

```
make run
```

### Make all

Will use build and tag as described above. Pushing is not included (yet)

```
make
```

## Example:
* First build the container image:
```
make
```
* Request server certificates, generate client cerificates and configure the Postgres database server accordingly
* Store serverca cert in `testdata/sererca.pem`
* Store client cert in `testdata/client_pgcdfga_chain.pem` and key in `testdata/client_pgcdfga.key` (in Kubernetes this will be secrets mounted as a volume)
* store ldap user / password in seperate files (in kubernetes this will be secrets mounted as a volume)
* configure `testdata/config.yaml` correctly (e.a. hostname, location of ldap user/pw files, etc.)
* start the container
```
docker run --rm -v $PWD:/pgcdfga_config pgcdfga
```

et voilà

# Developing
This tool was initially developed in-house by [bol.com](https://www.bol.com) and then open sourced.

## New minor release:
The very latest and greatest is always in master. So if we are working on 0.9.3, everything up until 0.9.2 is tagged. To tag 0.9.3, do the following:
* First run `make clean` (you will raise the version in `Dockerfile` after which `make clean` does not clean older versions)
* Freeze the current minor release by tagging it (on bol.com's internal gitlab, click `new tag`)
 * Also add a list of all patches of this minor release to the changelog
 * use `git log` if commit messages are properly formatted
 * use `git show` for a certain commit if you are unsure of one
 * use `git diff` if you are unsure of al (e.a. `git diff 0.9.2`)
* Finish by commiting the version change
 * Change the version of the package, which is listed in `pgcdfga/__init__.py` to reflect the new release
 * Use 'Raising version to [new_version]' for the commit message
 * This will be the first commit after freezing the previous version

Note: Up until 0.9.3 this procedure was not always followed thoroughly enough...

## New major release:
The very latest and greatest is always in master. So if we are currently working on major 0.9, every major up until 0.8 is a branch.
To start developing 1.0, you should create a 0.9 branch that points to the latest of 0.9 development, and after that change version info in master.
* First run `make clean` (you will raise the version in `Dockerfile` after which `make clean` does not clean older versions)
* Create a new branch from master
* Finish by commiting the version change
 * Change the version of the package, which is listed in `pgcdfga/__init__.py` to reflect the new release
 * Use 'Raising version to [new_version]' for the commit message
 * Commit this version change as a first commit in the new branch (using merge requests)
