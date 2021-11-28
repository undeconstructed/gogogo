
modules = comms client game gogame gogamebin server

gogame.plugin: .FORCE
	go build -o gogame.plugin ./gogamebin

server.run: gogame.plugin
	go run ./server

listgames:
	curl -v 'localhost:1235/api/games'

makegame:
	curl -XPOST -H"Content-Type: application/json" -v 'localhost:1235/api/games' --data '{"players":[{"name":"phil","colour":"red"}],"options":{"goal":8}}'

test: $(modules:=.test)

%.test: %
	go test ./$<

generate.grpc:
	protoc --go_out=. --go_opt=M --go_opt=paths=source_relative     --go-grpc_out=. --go-grpc_opt=paths=source_relative game/game.proto

.PHONY: .FORCE

.FORCE:
