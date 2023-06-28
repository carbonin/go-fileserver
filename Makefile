PWD = $(shell pwd)
LISTEN_PORT := $(or ${LISTEN_PORT}, 8000)

.PHONY: certs
certs:
	#openssl req -x509 -sha256 -nodes -days 365 -newkey rsa:2048 -keyout certs/tls.key -out certs/tls.crt -subj "/CN=localhost" -addext "subjectAltName = DNS:localhost"
	openssl req -x509 -sha256 -nodes -days 365 -newkey rsa:2048 -keyout certs/tls.key -out certs/tls.crt -subj "/CN=localhost"

run: certs
	LISTEN_PORT=$(LISTEN_PORT) \
	HTTPS_KEY_FILE=$(PWD)/certs/tls.key HTTPS_CERT_FILE=$(PWD)/certs/tls.crt \
	DATA_DIR=$(PWD)/data/ \
	go run main.go
