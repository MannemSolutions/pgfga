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

LDAP_DEFAULTS = {'servers': [], 'user': None, 'password': None, 'port': 636,
                 'ldapbasedn': 'OU=DC=example,DC=com', 'conn_retries': True}

from ldap3 import ServerPool, Server, Connection, SUBTREE

class LDAPConnectionException(Exception):
    '''
    This exception is raised on errors in the LDAPConnection class.
    '''
    pass

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

        try:
            if not ldapconfig['enabled']:
                #ldap is disabled. Disable further checking
                return
        except KeyError:
            pass

        try:
            for key in ['servers', 'user', 'password', 'port']:
                if not ldapconfig[key]:
                    raise LDAPConnectionException('ldapconnection init requires a value for {}\
                                                  '.format(key))
        except Exception as error:
            raise LDAPConnectionException('ldapconnection should be a dict with the correct \
                                           key=value pairs.', error)

    def connect(self):
        '''
        This methods connect to a(n) ldap server(s).
        '''
        try:
            if not self.__config['enabled']:
                print('ldap sync is disabled')
                return None
        except KeyError:
            pass
        if not self.__connection:
            ldapservers = [Server(ldap_server, port=self.__config['port'], use_ssl=True,
                                  connect_timeout=1) for ldap_server in
                           self.__config['servers']]
            serverpool = ServerPool(ldapservers, active=self.__config['conn_retries'], exhaust=True)
            self.__connection = Connection(serverpool, self.__config['user'],
                                           self.__config['password'], auto_bind=True)
        return self.__connection

    def ldap_grp_mmbrs(self, ldapbasedn=None, ldapfilter=None):
        '''
        This function is used to get a list of users in a ldap group
        '''
        try:
            if not self.__config['enabled']:
                print('ldap sync is disabled')
                return []
        except KeyError:
            pass

        if not ldapbasedn:
            ldapbasedn = self.__config['basedn']
        if not '(' in ldapfilter:
            try:
                filter_template = self.__config['filter_template']
            except KeyError:
                print('ldapfilter {} is without "(" and no filter_template is set'.format(ldapfilter))
                raise
            _ldapfilter = filter_template % ldapfilter
            print('Using {} for group {}'.format(_ldapfilter, ldapfilter))
            ldapfilter = _ldapfilter
        result_set = set()
        conn = self.connect()
        groups = conn.extend.standard.paged_search(search_base=ldapbasedn,
                                                   search_filter=ldapfilter,
                                                   search_scope=SUBTREE,
                                                   attributes=['memberUid'],
                                                   paged_size=5,
                                                   generator=True)
        for group in groups:
            members = [uid.decode() for uid in group['raw_attributes']['memberUid']]
            result_set |= set(members)
        result_set.discard('dummy')
        return sorted(result_set)
