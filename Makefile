.PHONY: swag
SWAG_API_DIR := api
swag_clean:
	rm ${SWAG_API_DIR}/*

swag: swag_clean
	swag init --output ${SWAG_API_DIR} --parseInternal --parseDependency

tidy:
	go mod tidy

.PHONY: build
build:
	go build .

test_run: build
	./aha_webserver 

clean:
	go clean 