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
from pgcdfga import pgcdfga


class DictWithDefaultsTest(unittest.TestCase):
    """
    Test the dict_with_defaults function.
    """
    def test_valid_dict_with_defaults(self):
        '''
        Test dict_with_defaults for normal functionality
        '''
        base = {'a': 'b', 'b': 'c', 1: 2, '1': '2'}
        default = {'a': 'c', 'c': 'd', 1: 3, 2: 3, '1': '3', '2': 3}
        expected = {'a': 'b', 'b': 'c', 'c': 'd', 1: 2, 2: 3, '1': '2', '2': 3}
        result1 = pgcdfga.dict_with_defaults(base, default)
        result2 = pgcdfga.dict_with_defaults(None, default)
        result3 = pgcdfga.dict_with_defaults(base, None)
        self.assertEqual(result1, expected)
        self.assertEqual(result2, default)
        self.assertEqual(result3, base)

    def test_invalid_dict_with_defaults(self):
        '''
        Test dict_with_defaults with error input functionality
        '''
        correct = {'a': 'c', 'c': 'd', 1: 3, 2: 3, '1': '3', '2': 3}
        listvalues = [1, 2, 3, 4]
        with self.assertRaises(TypeError):
            pgcdfga.dict_with_defaults(listvalues, correct)
        with self.assertRaises(TypeError):
            pgcdfga.dict_with_defaults(correct, listvalues)


class NonWordCharReTest(unittest.TestCase):
    """
    Test the Non Word Characters regular expression.
    """
    def test_valid_non_word_char_re(self):
        '''
        Test NON_WORD_CHAR_RE for matches
        '''
        self.assertEqual(pgcdfga.NON_WORD_CHAR_RE.search('123_!?').group(0), '_')
        self.assertEqual(pgcdfga.NON_WORD_CHAR_RE.search('abc!?_').group(0), '!')
        self.assertEqual(pgcdfga.NON_WORD_CHAR_RE.search('ABC?_!').group(0), '?')

    def test_invalid_non_word_char_re(self):
        '''
        Test NON_WORD_CHAR_RE for non-matches
        '''
        self.assertEqual(pgcdfga.NON_WORD_CHAR_RE.search('1234abcdABCD'), None)
