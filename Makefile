build:
	go build ./cmd/pgfga
debug:
	~/go/bin/dlv debug --headless --listen=:2345 --api-version=2 --accept-multiclient ./cmd/pgfga
run:
	./pgfga
