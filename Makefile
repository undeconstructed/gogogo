
modules = comms client game go-game/lib go-game/bin rummy-game/lib rummy-game/bin server

gogame.bin: .FORCE
	go build -o ./run/go/bin ./go-game/bin

gogame.data: .FORCE
	cp -r ./go-game/web ./run/go
	cp ./go-game/data.json ./run/go
	-mkdir ./run/go/bind
	-mkdir ./run/go/save

rummygame.bin: .FORCE
	go build -o ./run/rummy/bin ./rummy-game/bin

rummygame.data: .FORCE
	cp -r ./rummy-game/web ./run/rummy
	-mkdir ./run/rummy/bind
	-mkdir ./run/rummy/save

server.run: gogame.bin gogame.data rummygame.bin rummygame.data
	-rm ./run/*/bind/*.pipe
	go run ./server --games go,rummy

listgames:
	curl -v 'localhost:1235/api/games'

makegame:
	curl -XPOST -H"Content-Type: application/json" -v 'localhost:1235/api/games' --data '{"type":"go","players":[{"name":"phil","colour":"red"}],"options":{"goal":8}}'

test: $(modules:=.test)

%.test: %
	go test ./$<

generate.grpc:
	protoc --go_out=. --go_opt=M --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative game/game.proto

pstree:
	sh -c 'PID=$$(ps --no-headers -o pid -C go) ; pstree -p -T $$PID'

.PHONY: .FORCE

.FORCE:
