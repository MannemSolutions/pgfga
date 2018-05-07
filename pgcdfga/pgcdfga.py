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


#This re finds characters that are not a alphabetical letter / digit
NON_WORD_CHAR_RE = re.compile('[^0-9a-zA-Z]')

def process_users(pgconn: PGConnection, users: dict, ldapconnection: LDAPConnection):
    '''
    This function is a subfunction of main, that is used to process all user config.
    '''
    errorcount = 0
    for username, userconfig in users.items():
        logging.debug("Processing user %s", username)
        logging.debug("User config: %s", userconfig)
        try:
            # merge USER_DEFAULTS into this userconfig
            userconfig = dict_with_defaults(userconfig, USER_DEFAULTS)

            #set ensure
            ensure = userconfig['ensure'].lower()

            #set expiry
            expiry = userconfig['expiry']
            if expiry:
                #enhance expiry. Basically, you can set only a small portion
                #(like only year, or only year-month) and the rest will be appended.
                expiry = str(expiry)
                expiry = expiry+'2000-12-31 23:59:59'[len(expiry):]
                expiry = datetime.datetime.strptime(expiry, '%Y-%m-%d %H:%M:%S')
                #If expiry date has passed, remove account / group
                if datetime.datetime.now() > expiry:
                    ensure = 'absent'
                    logging.debug("User %s is expired", username)

            # Remove if ensure=absent
            if  ensure == 'absent':
                pgconn.droprole(username)
                logging.debug("Dropping user %s", username)
                continue

            auth = userconfig['auth'].lower()
            try:
                ldapbasedn = userconfig['ldapbasedn']
            except:
                ldapbasedn = None
            auth = NON_WORD_CHAR_RE.sub('', auth)
            if not auth in AUTH_ENUM:
                auth = 'client_cert'
            logging.debug("auth = %s", auth)

            logging.debug("Creating user/role %s", username)
            if auth == 'ldapgroup':
                #create ldap group with ldap users
                #For ldap group, we don't specify options on group, but rather on direct users.
                pgconn.createrole(username)
                try:
                    ldapfilter = userconfig['ldapfilter']
                except KeyError:
                    ldapfilter = username
                members = ldapconnection.ldap_grp_mmbrs(ldapbasedn=ldapbasedn,
                                                        ldapfilter=ldapfilter)
                for member in members:
                    logging.debug("creating member %s", member)
                    #For ldap group, we don't specify options on group, but rather on direct users.
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
    #Configuration file look up.
    with open(args.configfile) as configfile:
        configdata = yaml.load(configfile)

    #Configure logging
    if args.verbose:
        logging.basicConfig(level=logging.DEBUG)
    else:
        try:
            loglevel = configdata['general']['loglevel']

            if isinstance(loglevel, int):
                numeric_level = loglevel
            elif isinstance(loglevel, str):
                numeric_level = getattr(logging, loglevel.upper(), None)
                if not isinstance(numeric_level, int):
                    raise ValueError('Invalid log level: %s' % loglevel)
            logging.basicConfig(level=numeric_level)
        except (KeyError, AttributeError):
            pass
    logging.debug("Running as user %s (uid %s)", getpass.getuser(), os.getuid())
    logging.debug("Running with config file %s", args.configfile)

    try:
        ldapconfig = dict_with_defaults(configdata['ldap'], LDAP_DEFAULTS)
    except:
        ldapconfig = LDAP_DEFAULTS

    try:
        ldap_enabled = ldapconfig['enabled']
    except:
        ldap_enabled = True

    if ldap_enabled:
        try:
            if args.ldappwfile:
                ldappwfile = args.ldappwfile
            else:
                ldappwfile = configdata['ldap']['passwordfile']
        except (KeyError, AttributeError):
            ldappwfile = '~/ldap/password'
        ldappwfile = os.path.realpath(os.path.expanduser(ldappwfile))

        try:
            if args.ldapuserfile:
                ldapuserfile = args.ldapuserfile
            else:
                ldapuserfile = configdata['ldap']['userfile']
        except (KeyError, AttributeError):
            ldapuserfile = '~/.ldap/user'
        ldapuserfile = os.path.realpath(os.path.expanduser(ldapuserfile))

        if not ldapconfig['user']:
            ldapconfig['user'] = open(ldapuserfile).read().strip()
        if not ldapconfig['password']:
            ldapconfig['password'] = open(ldappwfile).read().strip()

    return configdata, ldapconfig

def main():
    '''
    This function runs the main part of the script.
    '''
    parsed_args = arguments()
    errorcount = 0

    while True:
        try:
            configdata, ldapconfig = config(parsed_args)
            try:
                strict = dict_with_defaults(configdata['strict'], STRICT_DEFAULTS)
            except KeyError:
                strict = copy(STRICT_DEFAULTS)

            pgconn = PGConnection(dsn_params=configdata['postgresql']['dsn'],
                                  strict_params=strict)
            ldapconn = LDAPConnection(ldapconfig)

            if pgconn.is_standby():
                raise Exception('Postgres ({}) cluster is standby'.format(pgconn.dsn()))

            if 'users' in configdata:
                try:
                    logging.debug("Processing users %s", configdata['users'])
                    errorcount += process_users(pgconn, configdata['users'], ldapconn)
                except Exception as error:
                    logging.exception(str(error))
            else:
                logging.debug("No user config set in configdata")
            if 'databases' in configdata:
                try:
                    logging.debug("Processing databases %s", configdata['databases'])
                    errorcount += process_databases(pgconn, configdata['databases'])
                except Exception as error:
                    logging.exception(str(error))
            else:
                logging.debug("No database config set in configdata")
            if 'roles' in configdata:
                try:
                    logging.debug("Processing roles %s", configdata['roles'])
                    errorcount += process_roles(pgconn, configdata['roles'])
                except Exception as error:
                    logging.exception(str(error))
            else:
                logging.debug("No database config set in configdata")

            if pgconn.strict_params['users']:
                logging.debug("Strictifying roles")
                try:
                    pgconn.strictifyroles()
                except Exception as error:
                    logging.exception(str(error))
            if pgconn.strict_params['databases']:
                logging.debug("Strictifying databases")
                try:
                    pgconn.strictifydatabases()
                except Exception as error:
                    logging.exception(str(error))
            if pgconn.strict_params['extensions']:
                logging.debug("Strictifying extensions")
                try:
                    pgconn.strictifyextensions()
                except Exception as error:
                    logging.exception(str(error))

            logging.info("Finished applying config")

        except:
            logging.exception('Error occurred while processing:')
            errorcount += 1
            #returncode is actually % 256, so if that is 0, add an additional 1
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
            logging.debug("Waiting for %s", str(delay))
            time.sleep(delay)
        else:
            break
    sys.exit(errorcount)
