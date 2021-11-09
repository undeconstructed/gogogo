
modules = comms client game gogame main server

server.run:
	go run ./main server

listgames:
	curl -v 'localhost:1235/api/games'

makegame:
	curl -XPOST -H"Content-Type: application/json" -v 'localhost:1235/api/games' --data '{"players":[{"name":"phil","colour":"red"}],"options":{"goal":8}}'

test: $(modules:=.test)

%.test: %
	go test ./$<
