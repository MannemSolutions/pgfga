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
import os
import tempfile
import logging
import unittest
import unittest.mock
from unittest.mock import patch
from psycopg2.sql import Composed, SQL, Identifier
from pgcdfga.pgconnection import PGConnection, PGConnectionException, STRICT_DEFAULTS


logging.disable(logging.CRITICAL)


class PGConnectionTest(unittest.TestCase):
    """
    Test the PGConnection Class.
    """
    def test_mocked_pg_connection_init(self):
        '''
        Test PGConnection.init for normal functionality
        '''
        sslkey = '~/sslkey'
        normalized_sslkey_path = os.path.expanduser(sslkey)
        sslkey_exists = os.path.exists(normalized_sslkey_path)
        _dummy, missingkeyfile = tempfile.mkstemp()
        os.remove(missingkeyfile)

        with unittest.mock.patch('psycopg2.connect') as mock_connect:
            if not sslkey_exists:
                with open(normalized_sslkey_path, 'w') as sslkey_file:
                    sslkey_file.write('blaat')

            mock_con = mock_connect.return_value
            mock_con.closed = False
            pgconn = PGConnection(dsn_params={'server': ['server1', 'server2'],
                                              'sslkey': missingkeyfile})
            pgconn.connect()
            pgconn = PGConnection(dsn_params={'server': ['server1', 'server2'], 'sslkey': sslkey})
            pgconn.connect()
            pgconn.connect()
            # Test with correct permissions
            pgconn = PGConnection(dsn_params={'server': ['server1', 'server2'], 'sslkey': sslkey})
            os.chmod(normalized_sslkey_path, 0o600)
            pgconn.connect()
            self.assertIsInstance(pgconn, PGConnection)
            if not sslkey_exists:
                os.remove(normalized_sslkey_path)
        expected_msg = 'Init PGConnection class with a dict of connection parameters'
        with self.assertRaises(PGConnectionException, msg=expected_msg):
            pgconn = PGConnection(dsn_params='')
        with self.assertRaises(PGConnectionException, msg=expected_msg):
            pgconn = PGConnection(dsn_params={})
        with self.assertRaises(PGConnectionException, msg=expected_msg):
            pgconn = PGConnection()

    def test_pg_connection_dsn(self):
        '''
        Test PGConnection.dsn for normal functionality
        '''
        input_data = {'a': 'b', 'c': 'd'}
        pgconn = PGConnection(dsn_params=input_data)
        result = pgconn.dsn()
        expected_result = 'a=b c=d'
        self.assertEqual(result, expected_result)

        input_data = {'a': 'a', 'b': 'b', 'sslkey': '~/dummypath'}
        result = pgconn.dsn(input_data)
        expected_regex = '^a=a b=b sslkey=.*/dummypath$'
        self.assertRegex(result, expected_regex, msg=None)

    def test_mocked_runsql(self):
        '''
        Test PGConnection.run_sql for normal functionality
        '''
        test_qry = "select datname, datdba from pg_database where datname in " \
                   "('postgres', 'template0')"
        query_header = [("datname",), ("datdba",)]
        query_faulty_header = [0, 1]
        query_result = [("template0", 10), ("postgres", 11)]
        expected_result = [{'datname': 'template0', 'datdba': 10},
                           {'datname': 'postgres', 'datdba': 11}]
        expected_connstr = 'server=server1 dbname=postgres'
        with unittest.mock.patch('psycopg2.connect') as mock_connect:
            mock_con = mock_connect.return_value
            mock_cur = mock_con.cursor.return_value
            mock_cur.description = query_header
            mock_cur.fetchall.return_value = query_result
            result = PGConnection(dsn_params={'server': 'server1'}).run_sql(test_qry)
            mock_connect.assert_called_with(expected_connstr)
            mock_cur.execute.assert_called_with(test_qry, None)
            self.assertEqual(result, expected_result)
            mock_cur.description = query_faulty_header
            result = PGConnection(dsn_params={'server': 'server1'}).run_sql(test_qry)
            self.assertIsNone(result)

        with unittest.mock.patch('psycopg2.connect') as mock_connect:
            mock_con = mock_connect.return_value
            mock_cur = mock_con.cursor.return_value
            mock_cur.execute.side_effect = PGConnectionException
            with self.assertRaises(PGConnectionException):
                result = PGConnection(dsn_params={'server': 'server1'}).run_sql(test_qry)

    def test_mocked_is_standby(self):
        '''
        Test PGConnection.is_standby for normal functionality
        '''
        query_header = [("recovery",)]
        with unittest.mock.patch('psycopg2.connect') as mock_connect:
            mock_con = mock_connect.return_value
            mock_cur = mock_con.cursor.return_value
            mock_cur.description = query_header
            for expected_result in [True, False]:
                mock_cur.fetchall.return_value = [(expected_result,)]
                result = PGConnection(dsn_params={'server': 'server1'}).is_standby()
                self.assertEqual(result, expected_result)

    def test_strict_option(self):
        '''
        Test PGConnection.strict_option() for keyerrors.
        '''
        pgcon = PGConnection(dsn_params={'server': 'server1'}, strict_params={})
        for key in STRICT_DEFAULTS:
            self.assertEqual(pgcon.strict_option(key), STRICT_DEFAULTS[key])

    def test_mocked_dropdb(self):
        '''
        Test PGConnection.dropdb for normal functionality
        '''
        query_header = [("datname",)]
        dropped_db = 'foobar'
        with unittest.mock.patch('psycopg2.connect') as mock_connect:
            mock_con = mock_connect.return_value
            mock_cur = mock_con.cursor.return_value
            mock_cur.description = query_header
            mock_cur.fetchall.return_value = [(dropped_db,)]

            # Test without strict should return False
            pgcon = PGConnection(dsn_params={'server': 'server1'})
            self.assertFalse(pgcon.dropdb(dropped_db))
            # Test with strict should return True if dropped and not else
            pgcon = PGConnection(dsn_params={'server': 'server1'},
                                 strict_params={'databases': True})
            self.assertTrue(pgcon.dropdb(dropped_db))
            expected_qry = Composed([SQL('DROP DATABASE '), Identifier(dropped_db)])
            mock_cur.execute.assert_called_with(expected_qry, None)
            mock_cur.fetchall.return_value = []
            self.assertFalse(pgcon.dropdb(dropped_db))

    def test_mocked_createdb(self):
        '''
        Test PGConnection.createdb for normal functionality
        '''
        dbname = 'foo'
        ownername = 'bar'
        with patch.object(PGConnection, 'grantrole') as mock_grantrole, \
                patch.object(PGConnection, 'createrole') as mock_createrole, \
                patch.object(PGConnection, 'run_sql') as mock_runsql:
            mock_grantrole.return_value = False
            mock_createrole.return_value = False
            mock_runsql.return_value = [{'schemaname': dbname}]
            pgcon = PGConnection(dsn_params={'server': 'server1'})
            self.assertFalse(pgcon.createdb(dbname, ownername))
            mock_createrole.assert_any_call(ownername)
            mock_grantrole.assert_any_call('opex', ownername)
            mock_grantrole.assert_any_call('readonly', dbname+'_readonly')
            mock_runsql.return_value = []
            self.assertTrue(pgcon.createdb(dbname))
            mock_grantrole.return_value = True
            mock_createrole.return_value = True
            self.assertTrue(pgcon.createdb(dbname))

    def test_mocked_droprole(self):
        '''
        Test PGConnection.droprole for normal functionality
        '''
        rolename = 'foobar'
        with patch.object(PGConnection, 'run_sql') as mock_runsql:
            # Test with strict should return True if dropped and not else
            for strict in [True, False]:
                pgcon = PGConnection(dsn_params={'server': 'server1'},
                                     strict_params={'users': strict})
                self.assertEqual(pgcon.droprole(rolename), strict)

            mock_runsql.return_value = [{'datname': 'postgres', 'owner': 'postgres'}]
            # Test with result on role query should return True
            pgcon = PGConnection(dsn_params={'server': 'server1'})
            self.assertTrue(pgcon.droprole(rolename))
            expected_qry = Composed([SQL('DROP ROLE '), Identifier(rolename)])
            mock_runsql.assert_called_with(expected_qry)
            mock_runsql.return_value = []
            self.assertFalse(pgcon.droprole(rolename))

    def test_mocked_createrole(self):
        '''
        Test PGConnection.createrole for normal functionality
        '''
        rolename = 'foo'
        options = ['superuser']
        invalid_options = ['invalid_option']
        expected_qrys = []
        with patch.object(PGConnection, 'run_sql') as mock_runsql:
            mock_runsql.return_value = [{'rolname': rolename}]
            pgcon = PGConnection(dsn_params={'server': 'server1'})
            self.assertFalse(pgcon.createrole(rolename))
            mock_runsql.return_value = []
            self.assertTrue(pgcon.createrole(rolename, options))
            expected_qrys.append(Composed([SQL('CREATE ROLE '), Identifier(rolename)]))
            expected_qrys.append(Composed([SQL('ALTER ROLE '), Identifier(rolename),
                                           SQL(' WITH SUPERUSER')]))
            for expected_qry in expected_qrys:
                mock_runsql.assert_any_call(expected_qry)
            with self.assertRaises(PGConnectionException):
                pgcon.createrole(rolename, invalid_options)

    def test_mocked_setpassword(self):
        '''
        Test PGConnection.setpassword for normal functionality
        '''
        rolename = 'foo'
        md5password = 'md5'+'a'*32
        normal_password = '12345'
        with patch.object(PGConnection, 'run_sql') as mock_runsql:
            mock_runsql.return_value = [{'usename': rolename}]
            pgcon = PGConnection(dsn_params={'server': 'server1'})
            self.assertTrue(pgcon.setpassword(rolename, normal_password))
            self.assertTrue(pgcon.setpassword(rolename, md5password))
            expected_qry = Composed([SQL('alter user '), Identifier(rolename),
                                     SQL(' with encrypted password %s')])
            mock_runsql.assert_any_call(expected_qry, [md5password])

            mock_runsql.return_value = []
            self.assertFalse(pgcon.setpassword(rolename, md5password))

    def test_mocked_resetpassword(self):
        '''
        Test PGConnection.createrole for normal functionality
        '''
        rolename = 'foo'
        with patch.object(PGConnection, 'run_sql') as mock_runsql:
            pgcon = PGConnection(dsn_params={'server': 'server1'})
            mock_runsql.return_value = [{'usename': rolename}]
            self.assertTrue(pgcon.resetpassword(rolename))
            mock_runsql.return_value = []
            self.assertFalse(pgcon.resetpassword(rolename))
            expected_qry = Composed([SQL('alter user '), Identifier(rolename),
                                     SQL(' with password NULL')])
            mock_runsql.assert_any_call(expected_qry)

    def test_mocked_grantrole(self):
        '''
        Test PGConnection.grantrole for normal functionality
        '''
        granted = 'foo'
        grantee = 'bar'
        with patch.object(PGConnection, 'createrole') as mock_createrole, \
                patch.object(PGConnection, 'run_sql') as mock_runsql:
            pgcon = PGConnection(dsn_params={'server': 'server1'})
            mock_createrole.return_value = False
            mock_runsql.return_value = [{'granted_role': granted, 'grantee_role': grantee}]
            self.assertFalse(pgcon.grantrole(granted, grantee))
            mock_createrole.return_value = True
            mock_runsql.return_value = [{'granted_role': granted, 'grantee_role': grantee}]
            self.assertTrue(pgcon.grantrole(granted, grantee))
            mock_createrole.return_value = False
            mock_runsql.return_value = []
            self.assertTrue(pgcon.grantrole(granted, grantee))
            mock_createrole.assert_any_call(grantee)
            mock_createrole.assert_any_call(granted)

    def test_mocked_revokerole(self):
        '''
        Test PGConnection.revokerole for normal functionality
        '''
        rolename = 'foo'
        username = 'bar'
        with patch.object(PGConnection, 'run_sql') as mock_runsql:
            pgcon = PGConnection(dsn_params={'server': 'server1'})
            mock_runsql.return_value = [{'rolename': rolename}]
            self.assertTrue(pgcon.revokerole(username, rolename))
            mock_runsql.return_value = []
            self.assertFalse(pgcon.revokerole(username, rolename))
            expected_qry = Composed([SQL('REVOKE '), Identifier(rolename),
                                     SQL(' FROM '), Identifier(username)])
            mock_runsql.assert_any_call(expected_qry)

    def test_mocked_strify_roles(self):
        '''
        Test PGConnection.strictifyroles for normal operation
        '''
        empty_testset = []
        normal_testset = []
        qry1 = []
        qry2 = []
        normal_testset.append(qry1)
        normal_testset.append(qry2)
        qry1.append({'grantee': 'john'})
        qry2.append({'rolname': 'dba'})
        qry2.append({'rolname': 'operator'})

        with patch.object(PGConnection, 'run_sql') as mock_runsql, \
                patch.object(PGConnection, 'droprole') as mock_droprole, \
                patch.object(PGConnection, 'createrole') as mock_createrole, \
                patch.object(PGConnection, 'revokerole') as mock_revokerole:
            pgcon = PGConnection(dsn_params={'server': 'server1'})
            mock_droprole.return_value = True
            mock_revokerole.return_value = True
            mock_createrole.return_value = True
            pgcon.grantrole('scot', 'dba')
            mock_runsql.side_effect = empty_testset
            self.assertFalse(pgcon.strictifyroles())
            mock_runsql.side_effect = normal_testset
            self.assertTrue(pgcon.strictifyroles())
            mock_droprole.assert_any_call('operator')
            mock_revokerole.assert_any_call('john', 'dba')

    def test_mocked_strify_databases(self):
        '''
        Test PGConnection.strictifydatabases for normal operation
        '''
        with patch.object(PGConnection, 'run_sql') as mock_runsql, \
                patch.object(PGConnection, 'createrole') as mock_createrole, \
                patch.object(PGConnection, 'grantrole') as mock_grantrole, \
                patch.object(PGConnection, 'dropdb') as mock_dropdb:
            pgcon = PGConnection(dsn_params={'server': 'server1'})
            mock_dropdb.return_value = True
            mock_createrole.return_value = True
            mock_grantrole.return_value = True
            sql_return = []
            mock_runsql.return_value = sql_return
            pgcon.createdb('test1')
            sql_return.append({'datname': 'postgres'})
            sql_return.append({'datname': 'test1'})
            self.assertFalse(pgcon.strictifydatabases())
            sql_return.append({'datname': 'test2'})
            # This line will throw a keyerror on parsing return of run_sql
            sql_return.append({'keyError': 'This throws a'})
            self.assertTrue(pgcon.strictifydatabases())
            mock_dropdb.assert_any_call('test2')

    def test_mocked_strify_extensions(self):
        '''
        Test PGConnection.strictifyextensions for normal operation
        '''
        with patch.object(PGConnection, 'run_sql') as mock_runsql, \
                patch.object(PGConnection, 'createrole') as mock_createrole, \
                patch.object(PGConnection, 'grantrole') as mock_grantrole, \
                patch.object(PGConnection, 'dropextension') as mock_dropextension:
            pgcon = PGConnection(dsn_params={'server': 'server1'})
            mock_dropextension.return_value = True
            mock_createrole.return_value = True
            mock_grantrole.return_value = True
            sql_return = []
            mock_runsql.return_value = sql_return
            pgcon.createdb('test1')
            # at this state, self.__extensions[dbname] will throw a keyerror
            self.assertFalse(pgcon.strictifyextensions())
            pgcon.createextension('extension1', 'test1')
            sql_return.append({'extname': 'extension1'})
            self.assertFalse(pgcon.strictifyextensions())
            sql_return.append({'extname': 'extension2'})
            # This line will throw a keyerror on parsing return of run_sql
            sql_return.append({'keyError': 'This throws a'})
            self.assertTrue(pgcon.strictifyextensions())
            mock_dropextension.assert_any_call('extension2', 'test1')

    def test_mocked_dropextension(self):
        '''
        Test PGConnection.dropextension for normal functionality
        '''
        created_in_db = 'foo'
        extension_name = 'bar'
        with patch.object(PGConnection, 'run_sql') as mock_runsql:
            pgcon = PGConnection(dsn_params={'server': 'server1'})
            mock_runsql.return_value = []
            self.assertFalse(pgcon.dropextension(extension_name, created_in_db))
            mock_runsql.return_value = [{'datname': created_in_db}]
            self.assertTrue(pgcon.dropextension(extension_name, created_in_db))
            expected_qry = Composed([SQL('DROP EXTENSION IF EXISTS '),
                                     Identifier(extension_name)])
            mock_runsql.assert_any_call(expected_qry, database=created_in_db)
            for strict in [True, False]:
                pgcon = PGConnection(dsn_params={'server': 'server1'},
                                     strict_params={'extensions': strict})
                self.assertEqual(pgcon.dropextension(extension_name, created_in_db),
                                 strict)

    def test_mocked_createextension(self):
        '''
        Test PGConnection.createextension for normal functionality
        '''
        created_in_db = 'foo'
        extension_name = 'bar'
        create_in_schema = 'schema1'
        extension_version = '1.2.3.4'
        with patch.object(PGConnection, 'run_sql') as mock_runsql:
            pgcon = PGConnection(dsn_params={'server': 'server1'})
            mock_runsql.return_value = [{'extname': created_in_db}]
            self.assertFalse(pgcon.createextension(extension_name, dbname=created_in_db,
                                                   schemaname=create_in_schema,
                                                   version=extension_version))
            mock_runsql.return_value = []
            self.assertTrue(pgcon.createextension(extension_name, dbname=created_in_db,
                                                  schemaname=create_in_schema,
                                                  version=extension_version))
            expected_qry_create = Composed([SQL('CREATE EXTENSION IF NOT EXISTS '),
                                            Identifier(extension_name)])
            expected_qry_schema = Composed([SQL('SCHEMA '), Identifier(create_in_schema)])
            expected_qry_version = Composed([SQL('VERSION '), Identifier(extension_version)])
            expected_qry = SQL(' ').join([expected_qry_create, expected_qry_schema,
                                          expected_qry_version])

            mock_runsql.assert_any_call(expected_qry, database=created_in_db)
            self.assertTrue(pgcon.createextension(extension_name, dbname=created_in_db))
            mock_runsql.assert_called_with(SQL(' ').join([expected_qry_create]),
                                           database=created_in_db)
