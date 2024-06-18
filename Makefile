build:
	go build -o ./irrigation-system main.go config.go log.go water.go weather.go

test:
	go test -v

restart: build restartsrv

restartsrv:
	sudo systemctl restart irrigation.service

status:
	sudo systemctl status irrigation.service