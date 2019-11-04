#!/usr/bin/env python

# Copyright 2019 Bol.com
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

'''
Script that creates databases, users, extensions and roles from a
yaml config file / ldap

=== Authors
Sebastiaan Mannem <smannem@bol.com>
Jing Rao <jrao@bol.com>
'''

from copy import copy
from argparse import ArgumentParser
import logging
import sys
import os
import datetime
import re
import time
import getpass
import yaml
from pgcdfga.ldapconnection import LDAPConnection, LDAP_DEFAULTS
from pgcdfga.pgconnection import PGConnection, DB_DEFAULTS, EXTENSION_DEFAULTS, \
    ROLE_DEFAULTS, USER_DEFAULTS, STRICT_DEFAULTS


def dict_with_defaults(data=None, default=None):
    '''
    This function returns a new dictionary with key/values from a defaults dictionary,
    which are overwritten by key/values from a data dictionary.
    '''
    data = data or {}
    default = default or {}
    if not isinstance(data, dict):
        raise TypeError('dict_with_defaults expects data to be a dictionary')
    if not isinstance(default, dict):
        raise TypeError('dict_with_defaults expects default to be a dictionary')
    ret = {}
    ret.update(default)
    ret.update(data)
    return ret


AUTH_ENUM = ['ldapgroup', 'ldapuser', 'password', 'md5', 'clientcert']

LOG_LEVEL_ENUM = {'CRITICAL': logging.CRITICAL,
                  'ERROR': logging.ERROR,
                  'WARNING': logging.WARNING,
                  'INFO': logging.INFO,
                  'DEBUG': logging.DEBUG,
                  'NOTSET': logging.NOTSET}


# This re finds characters that are not a alphabetical letter / digit
NON_WORD_CHAR_RE = re.compile('[^0-9a-zA-Z]')


def process_user(pgconn: PGConnection, username: str, userconfig: dict,
                 ldapconnection: LDAPConnection):
    '''
    This function is a subfunction of process_users, that is used to process config for a user.
    '''
    # merge USER_DEFAULTS into this userconfig
    userconfig = dict_with_defaults(userconfig, USER_DEFAULTS)
    # set ensure
    ensure = userconfig['ensure'].lower()

    # set expiry
    expiry = userconfig['expiry']
    if expiry:
        # enhance expiry. Basically, you can set only a small portion
        # (like only year, or only year-month) and the rest will be appended.
        expiry = str(expiry)
        expiry = expiry + '2000-12-31 23:59:59'[len(expiry):]
        expiry = datetime.datetime.strptime(expiry, '%Y-%m-%d %H:%M:%S')
        # If expiry date has passed, remove account / group
        if datetime.datetime.now() > expiry:
            ensure = 'absent'
            logging.info("User %s is expired", username)

    # Remove if ensure=absent
    if ensure == 'absent':
        pgconn.droprole(username)
        logging.debug("Dropping user %s", username)
        return

    auth = userconfig['auth'].lower()
    try:
        ldapbasedn = userconfig['ldapbasedn']
    except KeyError:
        ldapbasedn = None
    auth = NON_WORD_CHAR_RE.sub('', auth)
    if auth not in AUTH_ENUM:
        auth = 'client_cert'
    logging.debug("auth = %s", auth)

    logging.debug("Creating user/role %s", username)
    if auth == 'ldapgroup':
        # create ldap group with ldap users
        # For ldap group, we don't specify options on group, but rather on direct users.
        pgconn.createrole(username)
        ldapfilter = userconfig.get('ldapfilter', username)
        prefix = userconfig.get('prefix', ldapconnection.get_param('prefix', ''))
        suffix = userconfig.get('suffix', ldapconnection.get_param('suffix', ''))
        template = prefix+'{0}'+suffix
        members = ldapconnection.ldap_grp_mmbrs(ldapbasedn=ldapbasedn,
                                                ldapfilter=ldapfilter, template=template)
        for member in members:
            logging.info("Creating member %s from LDAP group %s", member, username)
            # For ldap group, we don't specify options on group, but rather on direct users.
            pgconn.createrole(member, ['LOGIN'] + userconfig['options'])
            pgconn.grantrole(member, username)
            logging.debug("Resetting password for member %s", member)
            pgconn.resetpassword(member)
    else:
        pgconn.createrole(username, ['LOGIN'] + userconfig['options'])

    if auth in ['ldapuser', 'clientcert', 'ldapgroup']:
        logging.debug("Resetting password for user %s", username)
        pgconn.resetpassword(username)
    else:
        if userconfig['password']:
            pgconn.setpassword(username, userconfig['password'])

    for role in userconfig['memberof']:
        logging.debug("Granting %s to %s", role, username)
        pgconn.grantrole(username, role)


def process_users(pgconn: PGConnection, users: dict, ldapconnection: LDAPConnection):
    '''
    This function is a subfunction of main, that is used to process all user config.
    '''
    errorcount = 0
    for username, userconfig in users.items():
        logging.debug("Processing user %s", username)
        logging.debug("User config: %s", userconfig)
        try:
            process_user(pgconn, username, userconfig, ldapconnection)
        except Exception as error:
            pgconn.strict_params['users'] = False
            logging.exception(str(error))
            errorcount += 1
    return errorcount


def process_databases(pgconn: PGConnection, databases: dict):
    '''
    This function is a subfunction of main, that is used to process all database config.
    '''
    errorcount = 0
    for dbname, dbconfig in databases.items():
        logging.debug("Processing database %s", dbname)
        try:
            # merge USER_DEFAULTS into this databaseconfig
            dbconfig = dict_with_defaults(dbconfig, DB_DEFAULTS)
            if dbconfig['ensure'] == 'absent':
                logging.debug("Dropping database %s", dbname)
                pgconn.dropdb(dbname)
            else:
                logging.debug("Creating database %s", dbname)
                pgconn.createdb(dbname, dbconfig['owner'])
        except Exception as error:
            pgconn.strict_params['databases'] = False
            logging.exception(str(error))
            errorcount += 1
        for extname, extconfig in dbconfig['extensions'].items():
            try:
                # merge USER_DEFAULTS into this databaseconfig
                extconfig = dict_with_defaults(extconfig, EXTENSION_DEFAULTS)
                if extconfig['ensure'] == 'absent':
                    logging.debug("Dropping extension %s from database %s", extname, dbname)
                    pgconn.dropextension(extname, dbname)
                else:
                    logging.debug("Creating extension %s in database %s", extname, dbname)
                    schema = extconfig['schema']
                    version = extconfig['version']
                    pgconn.createextension(extname, dbname, schema, version)
            except Exception as error:
                pgconn.strict_params['extensions'] = False
                logging.exception(str(error))
                errorcount += 1
    return errorcount


def process_replication_slots(pgconn: PGConnection, replication_slots: list):
    '''
    This function is a subfunction of main, that is used to process all replication slot config.
    '''
    errorcount = 0
    for replication_slot in replication_slots:
        logging.debug("Processing replication slot %s", replication_slot)
        try:
            logging.debug("Creating role %s", replication_slot)
            pgconn.create_replication_slot(replication_slot)
        except Exception as error:
            logging.exception(str(error))
            errorcount += 1
    for replication_slot in list(set(pgconn.replication_slots()) - set(replication_slots)):
        try:
            logging.debug("Cluster contains replication slot that isn't in the config %s",
                          replication_slot)
            pgconn.drop_replication_slot(replication_slot)
        except Exception as error:
            logging.exception(str(error))
            errorcount += 1
    return errorcount


def process_roles(pgconn: PGConnection, roles: dict):
    '''
    This function is a subfunction of main, that is used to process all role config.
    '''
    errorcount = 0
    for rolename, roleconfig in roles.items():
        logging.debug("Processing role %s", rolename)
        try:
            # merge USER_DEFAULTS into this databaseconfig
            roleconfig = dict_with_defaults(roleconfig, ROLE_DEFAULTS)
            if roleconfig['ensure'] == 'absent':
                logging.debug("Dropping role %s", rolename)
                pgconn.droprole(rolename)
            else:
                logging.debug("Creating role %s", rolename)
                pgconn.createrole(rolename, roleconfig['options'])
                for parent in roleconfig['memberof']:
                    logging.debug("Granting role %s to %s", parent, rolename)
                    pgconn.grantrole(rolename, parent)
        except Exception as error:
            pgconn.strict_params['users'] = False
            logging.exception(str(error))
            errorcount += 1
    return errorcount


def arguments():
    '''
    This function collects all config and initializes all objects.
    '''
    parser = ArgumentParser(description="Script to create users, databases, \
    extensions and roles in a postgresql database according to ldap roles / yaml config")
    parser.add_argument("-c", "--configfile", default=os.path.expanduser('~/config/config.yaml'),
                        help='The config file to use')
    parser.add_argument("-u", "--ldapuserfile", default=None,
                        help='kube secret that holds pgldap postgres user')
    parser.add_argument("-p", "--ldappwfile", default=None,
                        help='kube secret that holds pgldap postgres password')
    parser.add_argument("-v", "--verbose", action='store_true',
                        help='Be more verbose')
    parser.add_argument("-d", "--rundelay", type=int, default=0,
                        help='Be more verbose')
    args = parser.parse_args()

    return args


def config(args):
    '''
    This function reads and returns config data
    '''
    # Configuration file look up.
    with open(args.configfile) as configfile:
        configdata = yaml.load(configfile)

    if 'ldap' not in configdata:
        configdata['ldap'] = {}
    if 'enabled' not in configdata['ldap']:
        configdata['ldap']['enabled'] = True
    if args.ldappwfile:
        configdata['ldap']['passwordfile'] = args.ldappwfile
    elif 'passwordfile' not in configdata['ldap']:
        configdata['ldap']['passwordfile'] = '~/ldap/password'
    if args.ldapuserfile:
        configdata['ldap']['userfile'] = args.ldapuserfile
    elif 'userfile' not in configdata['ldap']:
        configdata['ldap']['userfile'] = '~/.ldap/user'

    # Configure logging
    logformat = '%(asctime)s %(levelname)s: %(message)s'
    if args.verbose:
        logging.basicConfig(level=logging.DEBUG, format=logformat)
    else:
        try:
            loglevel = configdata['general']['loglevel']

            if isinstance(loglevel, int):
                numeric_level = loglevel
            elif isinstance(loglevel, str):
                numeric_level = getattr(logging, loglevel.upper(), None)
                if not isinstance(numeric_level, int):
                    raise ValueError('Invalid log level: %s' % loglevel)
            logging.basicConfig(level=numeric_level, format=logformat)
        except (KeyError, AttributeError):
            pass

    logging.debug("Running as user %s (uid %s)", getpass.getuser(), os.getuid())
    logging.debug("Running with config file %s", args.configfile)
    return configdata


def config_ldap(configdata):
    '''
    This function reads and returns config data for ldap and return a ldap connection.
    '''
    try:
        ldapconfig = dict_with_defaults(configdata['ldap'], LDAP_DEFAULTS)
    except (KeyError, TypeError):
        ldapconfig = LDAP_DEFAULTS

    if ldapconfig['enabled']:
        if not ldapconfig['user']:
            ldapuserfile = os.path.realpath(os.path.expanduser(configdata['ldap']['userfile']))
            ldapconfig['user'] = open(ldapuserfile).read().strip()
        if not ldapconfig['password']:
            ldappwfile = os.path.realpath(os.path.expanduser(configdata['ldap']['passwordfile']))
            ldapconfig['password'] = open(ldappwfile).read().strip()

    return ldapconfig


def proces_fga(configdata, pgconn, ldapconn):
    '''
    This function is a helper function for main.
    '''
    errorcount = 0
    if 'users' in configdata:
        logging.debug("Processing users %s", configdata['users'])
        errorcount += process_users(pgconn, configdata['users'], ldapconn)
    else:
        logging.debug("No user config set in configdata")
    if 'databases' in configdata:
        logging.debug("Processing databases %s", configdata['databases'])
        errorcount += process_databases(pgconn, configdata['databases'])
    else:
        logging.debug("No database config set in configdata")
    if 'replication_slots' in configdata:
        logging.debug("Processing replication slots %s", configdata['replication_slots'])
        errorcount += process_replication_slots(pgconn, configdata['replication_slots'])
    if 'roles' in configdata:
        logging.debug("Processing roles %s", configdata['roles'])
        errorcount += process_roles(pgconn, configdata['roles'])
    else:
        logging.debug("No database config set in configdata")

    if pgconn.strict_params['users']:
        logging.debug("Strictifying roles")
        pgconn.strictifyroles()
    if pgconn.strict_params['databases']:
        logging.debug("Strictifying databases")
        pgconn.strictifydatabases()
    if pgconn.strict_params['extensions']:
        logging.debug("Strictifying extensions")
        pgconn.strictifyextensions()
    return errorcount


def main():
    '''
    This function runs the main part of the script.
    '''
    parsed_args = arguments()

    while True:
        errorcount = 0
        try:
            configdata = config(parsed_args)
            ldapconfig = config_ldap(configdata)
            try:
                strict = dict_with_defaults(configdata['strict'], STRICT_DEFAULTS)
            except KeyError:
                strict = copy(STRICT_DEFAULTS)

            pgconn = PGConnection(dsn_params=configdata['postgresql']['dsn'],
                                  strict_params=strict)
            ldapconn = LDAPConnection(ldapconfig)

            if pgconn.is_standby():
                raise Exception('Postgres ({}) cluster is standby'.format(pgconn.dsn()))

            errorcount += proces_fga(configdata, pgconn, ldapconn)

            logging.info("Finished applying config")

        except Exception:
            logging.exception('Error occurred while processing:')
            errorcount += 1
            # returncode is actually % 256, so if that is 0, add an additional 1
            if errorcount and not errorcount % 256:
                errorcount += 1

        try:
            if parsed_args.rundelay:
                delay = parsed_args.rundelay
            else:
                delay = configdata['general']['rundelay']
        except (KeyError, AttributeError, TypeError, UnboundLocalError):
            print('rundelay not set')
            break
        if delay > 0:
            logging.info("Waiting for %s seconds", str(delay))
            time.sleep(delay)
        else:
            break
    sys.exit(errorcount)
