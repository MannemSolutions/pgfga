# Config

## Definition

### Main configuration

The pgfga config is parsed as a yaml file which can be set with:
- the environment variable 'PGFGACONFIG'
- the `-c` commandline argument (precedence over the nvironment variable)
- defaults to /etc/pgfga/config.yml.

The file should only hold one yaml document. When multiple are parsed, the last one is used.
The config can set multiple entries:
- general, which can set
  - loglevel, which defaults to info, can be set to debug for more verbose output
  - run_delay, which can delay pgfga before it starts running, which is a convenience in docker-compose environments where all start running together. **Note** that without a unit (e.a. the 's' in '1s'), this is in nanoseconds!!!
- strict: This is a legacy option which might be added to v2 releases in future endeavors, but is not supported ATM.
- ldap, which can set the ldap connection options:
  - user: See [Ldap credentials](#Ldap_credentials) for more info
  - password: See [Ldap credentials](#Ldap_credentials) for more info
  - servers: this is a list strings and every string is a connect string for an ldap server (full connection strings e.a. ldap://127.0.0.1:389)
  - conn_retries: pgfga can retry a connection if it fails
- pg_dsn, a map with all connection details to connect to postgres.
   - **Note** that instead of configuring in this chapter, the [environment variables](https://www.postgresql.org/docs/current/libpq-envars.html) can also be used.
   - Options configured in this chapter take precedence over environment variables
- databases: See the chapter below on [Databases](#database_configuration)
- users: See the chapter below on [Users and Roles](#users_and_roles)
- roles: See the chapter below on [Users and Roles](#users_and_roles)
- replication slots: See the chapter below on [Replication slots](#replication_slots)

### Database configuration
The databases to be created can be set in a map where the key is the name of the database, and the value is the configuration.
For databases the following can be set:
- owner: This is to be the owner of the database.
  - [pgfga](https://github.com/MannemSolutions/pgfga) will create the owner even if not defined anywhere else
- state: Wether it should exist (default) or should not. See the [State](#state) chapter for more details.
- extensions: This is a map of extensions, where the key is the name and the value is the applicable configuration. See the [Extension configuration](#extension_configuration) chapter for more details.

### Extension configuration
Extensions are configuraed as part of the database where they should be installed.

**Note** that the version extension still needs to be installed as part of the rdbms software deployment.
Usually all [contrib](https://www.postgresql.org/docs/current/contrib.html) they usually are available, but other extensions (lik PostGIS) need too be installed before they can be managed by [pgfga](https://github.com/MannemSolutions/pgfga).

The extensions value for databases is a map where the key is the name and the value is the definition.
For extensions the following can be set:
  - schema: the schema where it should be created in. If it is already installed in another schema it will be moved.
  - state: Wether it should exist (default) or should not. See the [State](#state) chapter for more details.
  - version: the version of the extension to be installed. If it is already installed with another version it will be altered. **Note** that extensions usually can only be upgraded, not downgraded.

### Users and Roles

#### Distinction

Within PostgreSQL there is not much difference between Roles and Users.
Basically Users are Roles with a `LOGIN` option.
But when implementing Fine Grained Access, and looking at the way directories are implemented, there is a big distinction between Groups (which closely relate to Roles) and Users.
In [pgfga](https://github.com/MannemSolutions/pgfga) we have decided to pick a middleground, which means:
- Technically there is only an implementation for a Role, and a user is a Role with a `LOGIN` option.
- Within the configuration definition there is a distinction.
  - Roles can only be a member of other roles, and they can have [options](#role_options) and a [state](#state)
  - Users can also be a member of other roles, and they can have [options](#role_options) and a [state](#state)
    - Additionally you can set authentication options (like a password, expiry, etc.).
    - Furthermore a User can have an authentication method (`auth`).
    - The ldap implementation is a very specific implementation of the `auth: ldap-group` setting.

**Note** that (probably against expectations) ldap groups are not configured as roles, but as Users with the `auth` type 'ldap-group`. Main reason is that all other authentication types (`ldap-user`, `clientcert`, `password`, and `md5`) are types of users.

#### auth types
the following auth types can be set for a User:
- ldap-group: This setting enables [pgfga](https://github.com/MannemSolutions/pgfga) to read group info from an ldap and reflect it as Roles and Users in Postgres. This setting also requires configuring:
  - ldapbasedn: This specifies the base of the subtree in which the search is to be constrained. It should be set to the DN of the group that holds subgroups and memberUID's
  - ldapfilter: This option can be used to filter objects out of the search. Usually it can be set to `(objectclass=*)`, which means all objects...
- ldap-user: Is expected to do ldap authentication, which means no passwords / expiry in postgres
- clientcert: Is expected to use client certificates for authentication, which means no passwords / expiry in postgres (same implementation as `ldap-user`)
- password: Is expected to use a password for authentication. The following options can be set:
  - password:
    - The password can be md5 hashed (which has preference), or cleartext.
    - Unless md5 hash is detected, [pgfga](https://github.com/MannemSolutions/pgfga) will hash it before setting the password with an `ALTER ROLE` statement
    - Seting an emptystring for password will reset the password
  - expiry:
    - when set this will check the expiry date and alter when needed
    - when not set, the expiry date will be reset
- md5: Same implementation as `password`.

#### Examples
1: Getting ldap users from an ldap group:
```yaml
users:
  dbateam:
    auth: ldap-group
    ldapbasedn: 'cn=dba,ou=groups,dc=pgfga,dc=org'
    ldapfilter: '(objectclass=*)'
    memberof:
    - opex
    options:
    -  SUPERUSER
```
What it does: [pgfga](https://github.com/MannemSolutions/pgfga) will connect to ldap and
- create a ROLE called `dbateam`
- create a USER (with `LOGIN`) for all ldap users in the group / sub groups and GRANT `dbateam` to all those users:
  - ldap group with dn `cn=dba,ou=groups,dc=pgfga,dc=org`
  - all subgroups in the ldap group with dn `cn=dba,ou=groups,dc=pgfga,dc=org`
- grant `opex` to the ROLE `dbateam`
- set SUPERUSER for `dbateam`

2: Create a local backup user with a password:
```yaml
users:
  backup_user:
    auth: password
    password: bckpa$$w0rd
    options:
    -  REPLICATION
    memberof:
    - backup
```
What it does: [pgfga](https://github.com/MannemSolutions/pgfga) will create a ROLE `backup_user`, and:
- `backup_user` will have LOGIN (it is a user), and the `REPLICATION` option (as specified)
  - if the user exists, the other options will be unmodified (unmanaged)
- `backup_user` and `bckpa$$w0rd` will be hashed to form an md5 password, which will be checked and altered if needed.
- `backup_user` will become a member of `backup`

### Replication slots

In the current implementation, replication slots only can have a [state](#state), and [pgfga](https://github.com/MannemSolutions/pgfga) will only create or drop a Physical Replication Slot.
The slot is not immediately reserved, or temporary.

## Special values

### Ldap credentials
pgfga uses an object we call a crddential.
the credential can be used with ldap users and ldap passwords, and allows to directly set a password, or read from a file, and define if it is base64 encoded.
For a credential, the following can be set:
- value: Use this to set the credential value directly in the config file
- file: Use this to read the value from a file. **Note** that `value` takes precedence over `file`
- base64: Set to true to store as base64 encoded `value` or in `file, and have pgfga decode the value

### State
For all objects in postgres, there is an option to define the state.
State works similar to the way it is implemented in Puppet, and in some Ansible modules.
You can define `Present` (the default) or `Absent`. The name of the state is not case sensitive.
**Note** that state is not always reflected in subobjects.
As an example, setting `state: Absent` on an ldap group does not automatically remove all associated ldap accounts.
This might be where a future option sctrict could be helpful...

### Role options
Postgres allows for the following role options to be set:
- (NO)SUPERUSER
- (NO)CREATEROLE
- (NO)CREATEUSER
- (NO)INHERIT
- (NO)LOGIN
- (NO)REPLICATION

Within [pgfga](https://github.com/MannemSolutions/pgfga) these options are also implemented.
The exact implementation is that:
- There are only 6 actual options (`SUPERUSER`, `CREATEROLE`, `CREATEUSER`, `INHERIT`, `LOGIN`, `REPLICATION`).
- All 6 options can be true (without the NO prefix) or false (with the NO prefix)
- in the lists with options, later options negate earlier options
- the options are case insensitive (`SUPERUSER` is respected same as `SuperUser`, `superuser`, etc...)

As such, a USER (which has LOGIN by default) could have the following extra options set (in that order):
- SUPERUSER
- NOSUPERUSER
- NOINHERIT
- INHERIT
- NOLOGIN
In that case:
- `SUPERUSER` is negated by `NOSUPERUSER`, which is the end result
- `NOINHERIT` is negated by `INHERIT`, which is the end result
- `LOGIN` (default for users) is negated by `NOLOGIN`, which is the end result

As such, [pgfga](https://github.com/MannemSolutions/pgfga) will check (and set if needed) the `NOSUPERUSER`, `INHERIT` and `NOLOGIN` options.
Also **note** that the other options (`CREATEROLE`, `CREATEUSER` and `REPLICATION`) will not be checked and altered...

