build: clean
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o bin/gozap-macos-amd64 .
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/gozap-linux-amd64 .;
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o bin/gozap-windows-amd64.exe .;

clean:
	rm -rf bin