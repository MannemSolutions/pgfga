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

# Read docker info from the actual Dockerfile
IMAGE := $(shell awk '/IMAGE:/ {print $$3}' Dockerfile)
VERSION := $(shell cat pgcdfga/__init__.py | grep "^__version__" | awk '{print $$3}' | tr -d '"')
PROJECT := $(shell awk '/PROJECT:/ {print $$3}' Dockerfile)

all: clean build tag push
all-latest: clean build tag-latest push-latest

clean:
	rm -rf pgcdfga.egg-info/
	docker rmi ${IMAGE}:${VERSION} || echo Could not clean ${IMAGE}:${VERSION}
	docker rmi ${PROJECT}/${IMAGE}:${VERSION} || echo Could not clean ${PROJECT}/${IMAGE}:${VERSION}
	docker rmi ${PROJECT}/${IMAGE}:latest || echo Could not clean ${PROJECT}/${IMAGE}:latest

run:
	docker run --rm -t ${IMAGE}:${VERSION}

build: Dockerfile
	docker build -t ${IMAGE}:${VERSION} -f Dockerfile .

tag: tag-version tag-latest

tag-version:
	docker tag ${IMAGE}:${VERSION} ${PROJECT}/${IMAGE}:${VERSION}

tag-latest:
	docker tag ${IMAGE}:${VERSION} ${IMAGE}:latest
	docker tag ${IMAGE}:${VERSION} ${PROJECT}/${IMAGE}:latest

push: push-version push-latest

push-version:
	docker push ${PROJECT}/${IMAGE}:${VERSION} || echo Could not push ${PROJECT}/${IMAGE}:${VERSION}

push-latest:
	docker push ${PROJECT}/${IMAGE}:latest || echo Could not push ${PROJECT}/${IMAGE}:${VERSION}

test:
	flake8 .
	coverage run --source pgcdfga setup.py test
	coverage report -m
