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

from setuptools import setup, find_packages

install_requirements = [
    'pyyaml==3.12',
    'psycopg2-binary==2.7.4',
    'ldap3==2.4.1'
]

setup(
    name='pgcdfga',
    version='0.9.4',
    packages=find_packages(exclude=['contrib', 'docs', 'tests']),
    install_requires=install_requirements,
    entry_points={
        'console_scripts': [
            'pgcdfga=pgcdfga.pgcdfga:main',
        ]
    }
)
