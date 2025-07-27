SHELL=/bin/bash

# APP info
APP_NAME := coco
APP_VERSION := 1.0.0_SNAPSHOT
APP_CONFIG := $(APP_NAME).yml
APP_EOLDate := "2025-12-31T10:10:10Z"
APP_STATIC_FOLDER := .public
APP_STATIC_PACKAGE := public
APP_UI_FOLDER := ui
APP_PLUGIN_FOLDER := plugins
PREFER_MANAGED_VENDOR=fase

include ../framework/Makefile

build-web:
	(cd web && pnpm install &&  pnpm run build)

build-widget:
	(cd web/widgets/searchbox && pnpm install && pnpm run build:server)
	(cd web/widgets/fullscreen && pnpm install && pnpm run build:server)

build-all:
	(rm -rif .public/)
	make build-web
	make build-widget
	make build

check-deps:
	@echo "🔍 Checking module dependency hierarchy..."
	@go run tools/check-deps.go

lint-arch: check-deps
	@echo "✅ Architecture validation complete"
