build: clean
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o bin/gozap-macos-amd64 .

clean:
	rm -rf bin