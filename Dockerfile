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

# IMAGE:          pgcdfga
# PROJECT:        dockerhub.com/bol.com
# DESCRIPTION:    Enforces Postgres Container Deployments Fine Grained Accesscontrol
# TO_BUILD/TAG:   make
# TO_PUSH:        make push

FROM python:3

WORKDIR /usr/src/app

COPY pgcdfga /usr/src/app/pgcdfga/
COPY setup.cfg setup.py /usr/src/app/

RUN pip install --upgrade pip && pip install --no-cache-dir .

RUN groupadd -r -g 999 pgcdfga && useradd -m --no-log-init -r -g pgcdfga -u 999 pgcdfga
#&& mkdir ~pgcdfga/conf ~pgcdfga/.postgresql ~pgcdfga/.ldap_secrets && chown pgcdfga: ~pgcdfga/conf ~pgcdfga/.postgresql ~pgcdfga/.ldap_secrets && chmod 600 ~pgcdfga/conf ~pgcdfga/.postgresql ~pgcdfga/.ldap_secrets

USER 999

ENTRYPOINT ["pgcdfga"]
