serve-front: # Launch the frontend dev server
	cd cmd/serve/front && npm i && npm run dev

serve-docs: # Launch the docs dev server
	npm i && npm run docs:dev

serve-back: # Launch the backend API and creates an admin user if needed
	ADMIN_EMAIL=admin@example.com ADMIN_PASSWORD=admin LOG_LEVEL=debug go run main.go serve

test-front: # Launch the frontend tests
	cd cmd/serve/front && npm i && npm test && cd ../../..

test-back: # Launch the backend tests
	go vet ./...
	go test ./... --cover

test: test-front test-back # Launch every tests

ts: # Print the current timestamp, useful for migrations
	@date +%s

outdated: # Print direct dependencies and their latest version
	go list -v -u -m -f '{{if not .Indirect}}{{.}}{{end}}' all

build-front: # Build the frontend
	cd cmd/serve/front && npm i && npm run build && cd ../../..

build-back: # Build the backend
	go build -tags release -ldflags="-s -w" -o seelf

build: build-front build-back # Build the final binary for the current platform

build-docs: # Build the docs
	npm i && npm run docs:build

prepare-release: # Update the version.go file with the SEELF_VERSION env variable
	@sed -i".bak" "s/version = \".*\"/version = \"$(SEELF_VERSION)\"/g" cmd/version/version.go

release-docker: # Build and push the docker image
	@docker buildx build --platform linux/amd64,linux/arm64/v8 -t "yuukanoo/seelf:$(SEELF_VERSION)" -t yuukanoo/seelf:latest --push .