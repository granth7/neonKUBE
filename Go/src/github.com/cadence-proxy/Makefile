#------------------------------------------------------------------------------
# FILE:         Makefile
# CONTRIBUTOR:  John C Burns
# COPYRIGHT:    Copyright (c) 2016-2019 by neonFORGE, LLC.  All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

.PHONY: test bins clean
default: test-no-unix

# export to the path
export PATH := $(GOPATH)/bin:$(PATH)

# Project, test, source, and build paths
PROJECT_ROOT = github.com/cadence-proxy
BUILD_DIR := $(NF_ROOT)/Build
COMPRESSED_DIR := $(NF_ROOT)/Lib/Neon.Cadence/Resources
APP_DIR=./cmd/cadenceproxy
LOG_DIR=./test/logs

# test dirs
NEON_TEST_DIR := $(NF_ROOT)/Test/Test.Neon.Cadence/bin/Debug/netcoreapp3.0
MESSAGES_TEST_DIR=./internal/messages
TEST_DIRS := $(MESSAGES_TEST_DIR)

# Gather source code		
SRC := $(APP_DIR)/main.go

# programs in the /cmd dir to build
PROGS = cadenceproxy

# executable names
WIN := cadence-proxy.win.dll
WIN_HEADER := cadence-proxy.win.h
LINUX := cadence-proxy.linux.so
LINUX_HEADER := cadence-proxy.linux.h
OSX := cadence-proxy.osx.so
OSX_HEADER := cadence-proxy.osx.h

# Use dep to update and grab source
# for dependencies
dep-ensured:
	dep ensure

cadence-proxy-windows: dep-ensured $(ALL_SRC)
	env GOOS=windows GOARCH=amd64 CGO_ENABLED=1 go build -i -v -buildmode=c-shared -o bin/$(WIN) $(APP_DIR)/*.go

cadence-proxy-linux: dep-ensured $(ALL_SRC)
	env GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -i -v -buildmode=c-shared -o bin/$(LINUX) $(APP_DIR)/*.go

cadence-proxy-osx: dep-ensured $(ALL_SRC)
	env GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -i -v -buildmode=c-shared -o bin/$(OSX) $(APP_DIR)/*.go

cadenceproxy: cadence-proxy-windows \
	cadence-proxy-linux \
	cadence-proxy-osx

bins: cadenceproxy \

test-no-unix: cadence-proxy-windows
	rm -f test/messages.test
	rm -f $(LOG_DIR)/test.log
	echo $(TEST_DIRS)

	# test messages package
	go test -race -v -o test/messages.test -coverprofile $(LOG_DIR)/messages-cover.out $(MESSAGES_TEST_DIR) | tee -a $(LOG_DIR)/test.log
	go tool cover -html=$(LOG_DIR)/messages-cover.out -o $(LOG_DIR)/messages-cover.html

test-no-windows: cadence-proxy-linux \
	cadence-proxy-osx 
	
	rm -f test/messages.test
	rm -f $(LOG_DIR)/test.log
	echo $(TEST_DIRS)

	# test messages package
	go test -race -v -o test/messages.test -coverprofile $(LOG_DIR)/messages-cover.out $(MESSAGES_TEST_DIR) | tee -a $(LOG_DIR)/test.log
	go tool cover -html=$(LOG_DIR)/messages-cover.out -o $(LOG_DIR)/messages-cover.html


test-bins: bins
	rm -f test/messages.test
	rm -f $(LOG_DIR)/test.log
	echo $(TEST_DIRS)

	# test messages package
	go test -race -v -o test/messages.test -coverprofile $(LOG_DIR)/messages-cover.out $(MESSAGES_TEST_DIR) | tee -a $(LOG_DIR)/test.log
	go tool cover -html=$(LOG_DIR)/messages-cover.out -o $(LOG_DIR)/messages-cover.html
	
clean:
	rm -Rf bin/*
	
	# test files and logs
	rm -f test/messages.test
	rm -f $(LOG_DIR)/test.log
	rm -f $(LOG_DIR)/messages-cover.out
	rm -f $(LOG_DIR)/messages-cover.html

	# dll files
	rm -f $(BUILD_DIR)/$(WIN)
	rm -f $(BUILD_DIR)/$(LINUX)
	rm -f $(BUILD_DIR)/$(OSX)

	# header files
	rm -f $(BUILD_DIR)/$(WIN_HEADER)
	rm -f $(BUILD_DIR)/$(LINUX_HEADER)
	rm -f $(BUILD_DIR)/$(OSX_HEADER)

	# compressed test executables
	rm -f $(COMPRESSED_DIR)/$(WIN).gz
	rm -f $(COMPRESSED_DIR)/$(LINUX).gz
	rm -f $(COMPRESSED_DIR)/$(OSX).gz
	rm -f $(NEON_TEST_DIR)/cadence-proxy.exe
