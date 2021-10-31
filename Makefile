
modules = comms client game gogame main server

server.run:
	go run ./main server	

test: $(modules:=.test)

%.test: %
	go test ./$<
