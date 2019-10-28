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

"""
Script that creates databases, users, extensions and roles from a
yaml config file / ldap

=== Authors
Sebastiaan Mannem <smannem@bol.com>
Jing Rao <jrao@bol.com>
"""

import logging
from ldap3 import ServerPool, Server, Connection, SUBTREE, MOCK_SYNC, OFFLINE_SLAPD_2_4
from ldap3.core.exceptions import LDAPException

LDAP_DEFAULTS = {'servers': [], 'user': None, 'password': None, 'port': 636,
                 'ldapbasedn': 'OU=DC=example,DC=com', 'conn_retries': True}


class LDAPConnectionException(Exception):
    '''
    This exception is raised on errors in the LDAPConnection class.
    '''


class LDAPConnection():
    '''
    Init a new ldap connection
    '''
    def __init__(self, ldapconfig=None):
        '''
        This method initializes a ldap connection object.
        '''
        self.__config = ldapconfig
        self.__connection = None

        if not self.__config.get('enabled', True):
            return

        for key in ['servers', 'user', 'password']:
            if not self.__get_param(key):
                logging.error("ldapconnection requires a value for '%s'.", key)
                if self.__config.get('enabled', True):
                    logging.info("Disabling LDAP synchronisation due to missing configuration.")
                    self.__config['enabled'] = False

    def __get_param(self, key, default=None):
        '''
        This method is used to retrieve a param, or set a default
        '''
        try:
            return self.__config[key]
        except KeyError:
            return default

    def connect(self):
        '''
        This methods connect to a(n) ldap server(s).
        '''
        if not self.__get_param('enabled', True):
            return None

        mock_connection = self.mock_connect()
        if mock_connection:
            pass
        elif not self.__connection:
            ldapservers = [Server(ldap_server,
                                  port=self.__get_param('port', 636),
                                  use_ssl=self.__get_param('use_ssl', True),
                                  connect_timeout=1) for ldap_server in
                           self.__get_param('servers')]
            con_retries = self.__get_param('conn_retries', 1)
            serverpool = ServerPool(ldapservers,
                                    active=con_retries,
                                    exhaust=(con_retries > 0))
            logging.debug("Attempting to connect to LDAP servers: %s", serverpool.servers)
            try:
                self.__connection = Connection(serverpool,
                                               self.__get_param('user', ''),
                                               self.__get_param('password', ''),
                                               auto_bind=True)
                logging.debug("Successfully connected to LDAP servers")
            except LDAPException as error:
                logging.error("Unable to connect to LDAP servers: %s", str(error))
                # If we get here then self.__connection will be None,
                # but let's return it explicitly in case there is some
                # condition where that isn't the case.
                raise
        return self.__connection

    def mock_connect(self):
        '''
        This method checks if mocking is needed and if so, creates a mocked
        connection instead of a real connection to an ldap.
        '''
        if not self.__get_param('enabled', True):
            return None

        if self.__connection:
            return self.__connection

        mockdata = self.__get_param('mockdata', {})
        if not mockdata:
            return None

        my_fake_server = Server('my_fake_server', get_info=OFFLINE_SLAPD_2_4)
        connection = Connection(my_fake_server,
                                user='cn=my_user,ou=test,o=lab',
                                password='my_password',
                                client_strategy=MOCK_SYNC)

        for user, userconfig in mockdata.items():
            connection.strategy.add_entry(user, userconfig)
        connection.bind()
        logging.debug("Mocking the LDAP connection")
        self.__connection = connection
        return connection

    def ldap_grp_mmbrs(self, ldapbasedn=None, ldapfilter=None):
        '''
        This function is used to get a list of users in a ldap group
        '''
        if not self.__get_param('enabled', True):
            return []

        if not ldapbasedn:
            ldapbasedn = self.__get_param('basedn', '')
        if '(' not in ldapfilter:
            filter_template = self.__get_param('filter_template')
            if not filter_template:
                msg = 'ldapfilter {} is without "(" and no filter_template is set'
                raise LDAPConnectionException(msg.format(ldapfilter))
            _ldapfilter = filter_template % ldapfilter
            ldapfilter = _ldapfilter
        result_set = set()
        conn = self.connect()
        if conn is None:
            logging.info("No LDAP connection available to fetch groups members")
        else:
            groups = conn.extend.standard.paged_search(search_base=ldapbasedn,
                                                       search_filter=ldapfilter,
                                                       search_scope=SUBTREE,
                                                       attributes=['memberUid'],
                                                       paged_size=5,
                                                       generator=True)
            for group in groups:
                members = [uid.decode() for uid in group['raw_attributes']['memberUid']]
                result_set |= set(members)
            logging.debug("LDAP server returned the groups %s", sorted(result_set))
            result_set.discard('dummy')
        return sorted(result_set)
