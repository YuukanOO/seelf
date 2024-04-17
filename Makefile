serve-front: # Launch the frontend dev server
	cd cmd/serve/front && npm i && npm run dev

serve-docs: # Launch the docs dev server
	npm i && npm run docs:dev

serve-back: # Launch the backend API and creates an admin user if needed
	ADMIN_EMAIL=admin@example.com ADMIN_PASSWORD=admin LOG_LEVEL=debug go run main.go serve

test: # Launch every tests
	cd cmd/serve/front && npm i && npm test && cd ../../..
	go vet ./...
	go test ./... --cover

ts: # Print the current timestamp, useful for migrations
	@date +%s

outdated: # Print direct dependencies and their latest version
	go list -v -u -m -f '{{if not .Indirect}}{{.}}{{end}}' all

build: # Build the final binary for the current platform
	cd cmd/serve/front && npm i && npm run build && cd ../../..
	go build -ldflags="-s -w" -o seelf

build-docs: # Build the docs
	npm i && npm run docs:build

prepare-release: # Update the version.go file with the SEELF_VERSION env variable
	@sed -i".bak" "s/version = \".*\"/version = \"$(SEELF_VERSION)\"/g" cmd/version/version.go

release-docker: # Build and push the docker image
	@docker buildx build --platform linux/amd64,linux/arm64/v8 -t "yuukanoo/seelf:$(SEELF_VERSION)" -t yuukanoo/seelf:latest --push .