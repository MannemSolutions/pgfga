#! /usr/bin/env python

"""Enforce Fine Grained Access on a Postgres deployment."""

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
This module installs pgcdfga as a binary.
'''

import codecs
import os
import re
from setuptools import setup, find_packages

INSTALL_REQUIREMENTS = [
    'pyyaml==3.12',
    'psycopg2-binary==2.7.4',
    'ldap3==2.4.1'
]


def find_version():
    '''
    This function reads the pgcdfga version from pgcdfga/__init__.py
    '''
    here = os.path.abspath(os.path.dirname(__file__))
    with codecs.open(os.path.join(here, 'pgcdfga', '__init__.py'), 'r') as file_pointer:
        version_file = file_pointer.read()
    version_match = re.search(r"^__version__ = ['\"]([^'\"]*)['\"]",
                              version_file, re.M)
    if version_match:
        return version_match.group(1)
    raise RuntimeError("Unable to find version string.")


setup(
    name='pgcdfga',
    version=find_version(),
    packages=find_packages(exclude=['contrib', 'docs', 'tests']),
    install_requires=INSTALL_REQUIREMENTS,
    entry_points={
        'console_scripts': [
            'pgcdfga=pgcdfga.pgcdfga:main',
        ]
    }
)
