BINARY = tempsensorserver
PI_HOST = pi@192.168.188.151
PI_BIN = /usr/local/bin/$(BINARY)
SERVICE = $(BINARY).service

.PHONY: build test clean deploy

build:
	GOOS=linux GOARCH=arm GOARM=7 go build -o $(BINARY)-linux-arm .

test:
	go test ./... -v

clean:
	rm -f $(BINARY)-linux-arm

deploy: build
	scp $(BINARY)-linux-arm $(PI_HOST):$(PI_BIN)
	scp $(SERVICE) $(PI_HOST):/tmp/$(SERVICE)
	ssh $(PI_HOST) "sudo mv /tmp/$(SERVICE) /etc/systemd/system/ && sudo systemctl daemon-reload && sudo systemctl restart $(BINARY)"
	@echo "deployed and restarted on $(PI_HOST)"
