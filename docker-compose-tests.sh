#!/bin/bash
set -x
set -e
docker-compose up -d ldap postgres

docker-compose build --no-cache --force-rm
docker-compose up pgfga || exit 2
cat testdata/pgtester/tests.yaml | docker-compose run pgtester pgtester || exit 3

echo "All is as expected"
