# PGFGA - Postgres Fine Grained Access tool

Tool to configure and manage Postgres logical objects (Users, Roles, Databases and Extensions).
Users and roles can be synced from an ldap directory.

# License

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

# Requirements

## PGFGA config

PGFGA can be configured using a configuration yaml.
In container deployments (like Kubernetes), that might be configmap mounted as a volume.

### Postgres User account

The PGFGA tool requires a postgres user with access and SUPERUSER privilleges to run.
The user could be made available either with:
- Local access (when running inside of master container) and using ident (or trust) authentication
- A staged username / password e.a. configured with setup.sql script (by setting PG_PRIMARY_USER, PG_PRIMARY_PASSWORD, etc.).
- A staged pgfga user with client certificate authetnication
- Setting up ldap authentication for the postgres user

The most convenient way is to use a client certificate, handed out to PGCDFGA user as configured in config.yaml (postgres.dsn).

# Contributing
Please see [Developing](DEVELOP.md) for more information.
