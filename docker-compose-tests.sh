#!/bin/bash
set -x
set -e
docker-compose up -d ldap postgres

docker-compose build --no-cache --force-rm
docker-compose up pgfga || exit 2
docker-compose up pgtester pgtester | grep -q ERROR && exit 3

echo "All is as expected"
