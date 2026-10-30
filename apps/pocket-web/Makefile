SHELL := /bin/sh
FRONTEND := frontend
BACKEND := backend

.PHONY: setup dev backend frontend test build typecheck clean screenshots

setup:
	cd $(BACKEND) && go mod download
	cd $(FRONTEND) && bun install

dev: build
	@echo "Starting Pocket Web at http://127.0.0.1:8088"
	cd $(BACKEND) && POCKET_WEB_UI="$(CURDIR)/$(FRONTEND)/dist" go run .

backend:
	cd $(BACKEND) && POCKET_WEB_UI="$(CURDIR)/$(FRONTEND)/dist" go run .

frontend:
	cd $(FRONTEND) && npm run dev

test:
	cd $(BACKEND) && go test -race ./...
	cd $(FRONTEND) && npm test
	cd $(FRONTEND) && ./node_modules/typescript/bin/tsc --noEmit

build:
	cd $(BACKEND) && go build -o backend ./...
	cd $(FRONTEND) && ./node_modules/typescript/bin/tsc --noEmit
	cd $(FRONTEND) && bun build.mjs

typecheck:
	cd $(FRONTEND) && ./node_modules/typescript/bin/tsc --noEmit

screenshots:
	./capture-screenshots.sh

clean:
	rm -rf $(FRONTEND)/dist $(BACKEND)/backend
