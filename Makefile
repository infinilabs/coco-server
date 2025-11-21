SHELL=/bin/bash

# APP info
APP_NAME := coco
APP_VERSION := 1.0.0_SNAPSHOT
APP_CONFIG := $(APP_NAME).yml
APP_EOLDate := "2030-12-31T10:10:10Z"
APP_STATIC_FOLDER := .public
APP_STATIC_PACKAGE := public
APP_UI_FOLDER := ui
APP_PLUGIN_FOLDER := plugins
PREFER_MANAGED_VENDOR=fase

include ../framework/Makefile


init-web-env:
	#nvm use 18

build-web:
	(cd web && pnpm install &&  pnpm run build)

build-widget: init-web-env
	(cd web/widgets/searchbox && pnpm install . && pnpm run build:server)
	(cd web/widgets/fullscreen && pnpm install . && pnpm run build:server)

build-all: init-web-env
	(rm -rif .public/)
	make build-web
	make build-widget
	make build

int-test:
	./tests/assets/run_integration_tests.py