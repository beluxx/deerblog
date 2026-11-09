.PHONY: all build run css test clean

all: css build

css:
	npx tailwindcss -i ./web/css/input.css -o ./static/css/tailwind.min.css --minify

css-watch:
	npx tailwindcss -i ./web/css/input.css -o ./static/css/tailwind.css --watch

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="-s -w" -o ./bin/blog ./cmd/server

run:
	go run ./cmd/server/main.go

test:
	CGO_ENABLED=0 go test -v ./...

clean:
	rm -rf bin/ data/
