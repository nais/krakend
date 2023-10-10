APP=debugapp

release:
	go build -a -installsuffix cgo -o $(APP)

local:
	go run main.go --bind-address=127.0.0.1:8080