# PGFGA - Postgres Fine Grained Access tool

## License

   Copyright 2021 - Bol.com and Mannem Solution

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.

# Introduction

PGFGA is a tool to configure and manage Postgres logical objects (Users, Roles, Databases, Extensions and Replication Slots).
Users and roles can be synced from an ldap directory.

## The origin
Usually organizations manage their users in a directory (like AD, ldap, or a cloud IAM).
Although Postgres can integrate with such tools for authentication, it does not for authorization.
Furthermore, the users need to either be definied in postgres or be mapped to an existing user before authentication can finish succesfully.

As such many enterprises have looked into syncing autorization into postgres and many tools exist.
At [bol.com](www.bol.com) an existing tool based on python 2 was used.
For conformity, the tool was inhouse rebuilt into python 3 and after that enhanced with unittests, and extra capabilities.

For the bol.com usecase, it made sense to manage more types of objects next to users and roles.
As an example, managing ownership on a database means that the database should exist too.
The tool was expanded to manage databases, extensions and replication slots too.
And thus, pgcdfga was born, built, and maintained.

After a few years, Mannem Solutions has been offered the ability to adopt the solution.
We have renamed it to pgfga (PostGres Fine Grained Access), rebuilt it in GoLang and are now maintaining (and using) the new tool in our own solutions.

# Requirements

## PGFGA config

PGFGA can be configured using a configuration yaml.
In container environments (like Kubernetes), that could be a configmap mounted as a volume.
For more details on the configuration format, please refer to [our config description](CONFIG.md).

### Postgres User account

The PGFGA tool requires a postgres user with access and SUPERUSER privilleges to run.
The user could be made available either with:
- Local access (when running inside of master container) and using ident authentication
- A staged username / password e.a. configured with setup.sql script (by setting PG_PRIMARY_USER, PG_PRIMARY_PASSWORD, etc.).
- A staged pgfga user with client certificate authentication.
- Setting up ldap authentication for the postgres user (see the [docker-compose example](docker-compose.yaml) for an example of this setup).

# Downloading
The most straight forward option is to download the [pgfga](https://github.com/MannemSolutions/pgfga) binary directly from the [github release page](https://github.com/MannemSolutions/pgfga/releases).
But there are other options, like
- using the [container image from dockerhub](https://hub.docker.com/repository/docker/mannemsolutions/pgfga/general)
- direct build from source (if you feel you must)

Please refer to [our download instructions](DOWNLOAD_AND_RUN.md) for more details on all options.

# Using
**Note** please refer to our [configuration documentation](CONFIG.md) to learn about all capabilities and configration features of pgfga.

After downloading the binary to a folder in your path, you can run pgfga with a command like:
```bash
pgfga -c ./myconfig.yml
```

# Contributing
Please see [Developing](DEVELOP.md) for more information.
