current_dir = $(shell pwd)

build:
	docker run -e "CGO_ENABLED=0" -v $(current_dir):/go/bin --rm golang go get -v github.com/DonnchaC/oniongateway/entry_proxy
	docker build -t oniongateway-entryproxy .
	echo "Done! run with 'docker run oniongateway-entryproxy'"
