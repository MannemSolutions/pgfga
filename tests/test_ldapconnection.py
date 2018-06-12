#!/usr/bin/env python3

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
This module holds all unit tests for the pgcdfga module
'''
import unittest
from copy import copy
import ldap3
from pgcdfga.ldapconnection import LDAPConnectionException, LDAPConnection


class LDAPConnectionTest(unittest.TestCase):
    """
    Test the LDAPConnection Class.
    """
    def test_ldap_group_members_mocked(self):
        '''
        Test test_ldap_group_members_mocked for normal functionality
        Basically, this is happy flow and touches almost all code
        '''
        ldap_config = {}
        ldap_config['mockdata'] = mockdata = {}
        groupname = "cn=team7,OU=test,DC=example,DC=com"
        mockdata[groupname] = mockgroup = {}
        mockgroup["cn"] = ["team1"]
        mockgroup["description"] = ["Dev Team 1"]
        mockgroup["memberUid"] = groupmembers = []

        for userid in range(5):
            username = 'cn=user{0},ou=test,DC=example,DC=com'.format(userid)
            mockdata[username] = {'userPassword': 'test{0:04}'.format(userid),
                                  'sn': 'user{0}_sn'.format(userid)}
            groupmembers.append('user{0}'.format(userid))
        ldap_config['basedn'] = 'OU=test,DC=example,DC=com'
        ldap_config['servers'] = ['ldap.example.com']
        ldap_config['user'] = 'Nobody'
        ldap_config['password'] = 'Secret'
        ldap_config['filter_template'] = '(cn=%s)'

        ldap_con = LDAPConnection(ldap_config)
        self.assertIsInstance(ldap_con.connect(), ldap3.Connection)
        result = ldap_con.ldap_grp_mmbrs(ldapfilter='(cn=team1)')
        self.assertEqual(set(groupmembers), set(result))
        result = ldap_con.ldap_grp_mmbrs(ldapfilter='team1')
        self.assertEqual(set(groupmembers), set(result))

    def test_mocked_invalid_filter(self):
        '''
        Test test_mocked_invalid_filter without ldap filter and ldap filter template.
        '''
        ldap_config = {}
        ldap_config['mockdata'] = '1234'
        ldap_config['servers'] = ['ldap.example.com']
        ldap_config['user'] = 'Nobody'
        ldap_config['password'] = 'Secret'

        ldap_con = LDAPConnection(ldap_config)
        expected_msg = 'ldapfilter team1 is without "(" and no filter_template is set'
        with self.assertRaises(LDAPConnectionException,
                               msg=expected_msg):
            ldap_con.ldap_grp_mmbrs(ldapfilter='team1')

    def test_mocked_disabled(self):
        '''
        Test test_mocked_disabled tests the mocked connectionif ldapsync is disabled.
        '''
        ldap_config = {}
        ldap_config['mockdata'] = '1234'
        ldap_config['servers'] = ['ldap.example.com']
        ldap_config['user'] = 'Nobody'
        ldap_config['password'] = 'Secret'

        ldap_config['enabled'] = False
        ldap_con = LDAPConnection(ldap_config)
        self.assertIsNone(ldap_con.mock_connect())
        self.assertEqual(set(), set(ldap_con.ldap_grp_mmbrs()))

    def test_ldap_grp_mmbrs_method(self):
        '''
        Test test_ldap_grp_mmbrs_method for normal functionality.
        It actually tries to connect and breaks on that.
        '''
        ldap_config = {}
        ldap_config['basedn'] = 'OU=test,DC=example,DC=com'
        ldap_config['servers'] = ['127.0.0.1']
        ldap_config['user'] = 'Nobody'
        ldap_config['password'] = 'Secret'
        ldap_config['port'] = 1
        ldap_config['conn_retries'] = 0

        with self.assertRaises(ldap3.core.exceptions.LDAPSocketOpenError):
            LDAPConnection(ldap_config).connect()

    def test_ldap_disabled(self):
        '''
        Test LDAPConnection in disabled mode
        '''
        ldap_con = LDAPConnection({'enabled': False})
        self.assertIsNone(ldap_con.connect())
        result = ldap_con.ldap_grp_mmbrs(ldapfilter='(cn=team1)')
        self.assertEqual(set(), set(result))

    def test_ldap_missing_args(self):
        '''
        Test LDAPConnection without important arguments
        '''
        ldap_config = {}
        ldap_config['servers'] = ['ldap.example.com']
        ldap_config['user'] = 'Nobody'
        ldap_config['password'] = 'Secret'
        ldap_config['port'] = 1

        for key in ['servers', 'user', 'password']:
            mia_ldap_config = copy(ldap_config)
            del mia_ldap_config[key]
            with self.assertRaises(LDAPConnectionException,
                                   msg='ldapconnection init requires a value for {0}'.format(key)):
                LDAPConnection(mia_ldap_config)
