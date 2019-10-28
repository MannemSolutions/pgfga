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
PROJECTS ?= dockerhub.com/bol.com

all: clean test build tag push
all-latest: clean test build tag-latest push-latest
test: test-flake8 test-pylint test-coverage

clean:
	@docker rmi $(IMAGE):$(VERSION) || echo Could not clean $(IMAGE):$(VERSION)
	@$(foreach project, $(PROJECTS), docker rmi $(project)/$(IMAGE):$(VERSION) || echo Could not clean $(project)/$(IMAGE):$(VERSION))
	@$(foreach project, $(PROJECTS), docker rmi $(project)/$(IMAGE):latest || echo Could not clean $(project)/$(IMAGE):latest)

run:
	docker run --rm -t ${IMAGE}:${VERSION}

build: build-docker-image build-binary

build-binary:
	docker run -ti --rm --name pgcdfga_builder -v $$PWD:/host centos:7 /host/build_binary.sh

build-docker-image: Dockerfile
	docker build -t ${IMAGE}:${VERSION} -f Dockerfile .

build-test-container:
	docker build -t pgcdfga-test . -f Dockerfile-test

tag: tag-version tag-latest

tag-version:
	$(foreach project, $(PROJECTS), docker tag $(IMAGE):$(VERSION) $(project)/$(IMAGE):$(VERSION))

tag-latest:
	$(foreach project, $(PROJECTS), docker tag $(IMAGE):$(VERSION) $(project)/$(IMAGE):latest)

push: push-version push-latest

push-version:
	@$(foreach project, $(PROJECTS), docker push $(project)/$(IMAGE):$(VERSION) || echo Could not push $(project)/$(IMAGE):$(VERSION))

push-latest:
	@$(foreach project, $(PROJECTS), docker push $(project)/$(IMAGE):latest || echo Could not push $(project)/$(IMAGE):latest)

test-flake8:
	docker run -ti -v $$PWD:/host --rm --name pgcdfga_test pgcdfga-test:latest /bin/bash -c 'cd /host && flake8 .'

test-pylint:
	docker run -ti -v $$PWD:/host --rm --name pgcdfga_test pgcdfga-test:latest /bin/bash -c 'cd /host && pylint *.py pgcdfga tests'

test-coverage:
	docker run -ti -v $$PWD:/host --rm --name pgcdfga_test pgcdfga-test:latest /bin/bash -c 'cd /host && coverage run --source pgcdfga setup.py test ; coverage report -m'
