
modules = comms client game gogame main server

server.run:
	go run ./main server

creategameone:
	curl -v localhost:1235/create?name=one

test: $(modules:=.test)

%.test: %
	go test ./$<
