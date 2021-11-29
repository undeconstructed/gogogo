package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"

	"github.com/undeconstructed/gogogo/game"
	"github.com/undeconstructed/gogogo/gogame"
)

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	bind := os.Args[1]

	data := gogame.LoadJson(".")

	gsrvr, err := makeGSrv(bind, data)
	if err != nil {
		panic("cannot make gsrv")
	}

	rand.Seed(time.Now().Unix())

	grp, gctx := errgroup.WithContext(context.Background())

	grp.Go(func() error {
		return gsrvr.StartServer(gctx)
	})

	err = grp.Wait()
	if err != nil {
		panic("error")
	}
}

type gsrv struct {
	game.UnimplementedRGameServer

	listener net.Listener
	data     gogame.GameData

	id string
	gg game.Game
}

func makeGSrv(bind string, data gogame.GameData) (*gsrv, error) {
	l, err := net.Listen("tcp", bind)
	if err != nil {
		return nil, err
	}
	return &gsrv{
		listener: l,
		data:     data,
	}, nil
}

func (s *gsrv) StartServer(ctx context.Context) error {
	srv := grpc.NewServer()
	game.RegisterRGameServer(srv, s)

	return srv.Serve(s.listener)
}

func (s *gsrv) Load(ctx context.Context, req *game.RLoadRequest) (*game.RLoadResponse, error) {
	if s.gg != nil {
		return nil, status.Errorf(codes.AlreadyExists, "game already present")
	}

	f, err := os.Open("state-" + req.Id + ".json")
	if err != nil {
		log.Error().Err(err).Msg("cannot open state file")
		return nil, err
	}
	gg, err := gogame.NewFromSaved(s.data, f)
	if err != nil {
		log.Error().Err(err).Msg("cannot restore state")
		return nil, err
	}

	s.id = req.Id
	s.gg = gg

	sg := s.gg.GetGameState()
	st := s.gg.GetTurnState()

	return &game.RLoadResponse{
		State: game.WrapGameState(&sg),
		Turn:  game.WrapTurnState(&st),
	}, nil
}

func (s *gsrv) Init(ctx context.Context, req *game.RInitRequest) (*game.RInitResponse, error) {
	if s.gg != nil {
		return nil, status.Errorf(codes.AlreadyExists, "game already present")
	}

	goal := 4
	if g0, ok := req.Options["goal"]; ok {
		if g1, err := strconv.Atoi(g0); err != nil {
			// XXX - lost error
			goal = int(g1)
		}
	}

	gg := gogame.NewGame(s.data, goal)
	s.id = req.Id
	s.gg = gg
	s.saveGame()

	sg := s.gg.GetGameState()

	return &game.RInitResponse{
		State: game.WrapGameState(&sg),
	}, nil
}

func (s *gsrv) AddPlayer(ctx context.Context, in *game.RAddPlayerRequest) (*game.RAddPlayerResponse, error) {
	if s.gg == nil {
		panic("no game")
	}

	err := s.gg.AddPlayer(in.Name, in.Colour)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "%v", err)
	}

	sg := s.gg.GetGameState()

	return &game.RAddPlayerResponse{
		State: game.WrapGameState(&sg),
	}, nil
}

func (s *gsrv) Start(context.Context, *game.RStartRequest) (*game.RStartResponse, error) {
	if s.gg == nil {
		panic("no game")
	}

	st, err := s.gg.Start()
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "%v", err)
	}
	s.saveGame()

	sg := s.gg.GetGameState()

	return &game.RStartResponse{
		State: game.WrapGameState(&sg),
		Turn:  game.WrapTurnState(&st),
	}, nil
}

func (s *gsrv) Play(ctx context.Context, in *game.RPlayRequest) (*game.RPlayResponse, error) {
	if s.gg == nil {
		panic("no game")
	}

	res, err := s.gg.Play(in.Player, game.Command{
		Command: game.CommandString(in.Command),
		Options: in.Options,
	})
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "%v", err)
	}
	s.saveGame()

	rr, _ := json.Marshal(res.Response)

	sg := s.gg.GetGameState()

	return &game.RPlayResponse{
		Response: string(rr),
		News:     game.WrapChanges(res.News),
		State:    game.WrapGameState(&sg),
		Turn:     game.WrapTurnState(&res.Next),
	}, nil
}

func (s *gsrv) GetGameState(context.Context, *game.Empty) (*game.RGameState, error) {
	if s.gg == nil {
		panic("no game")
	}
	return nil, status.Errorf(codes.Unimplemented, "method GetGameState not implemented")
}

func (s *gsrv) GetTurnState(context.Context, *game.Empty) (*game.RTurnState, error) {
	if s.gg == nil {
		panic("no game")
	}
	return nil, status.Errorf(codes.Unimplemented, "method GetTurnState not implemented")
}

func (s *gsrv) Destroy(context.Context, *game.RDestroyRequest) (*game.RDestroyResponse, error) {
	if s.gg == nil {
		panic("no game")
	}

	s.wipeGame()
	s.gg = nil

	return &game.RDestroyResponse{}, nil
}

func (s *gsrv) saveFileName() string {
	return fmt.Sprintf("state-%s.json", s.id)
}

func (s *gsrv) saveGame() {
	if s.gg == nil {
		panic("no game")
	}
	outFile, err := os.Create(s.saveFileName())
	if err != nil {
		log.Error().Err(err).Msg("can't save")
		return
	}
	defer outFile.Close()

	s.gg.WriteOut(outFile)
}

func (s *gsrv) wipeGame() {
	if s.gg == nil {
		panic("no game")
	}
	err := os.Remove(s.saveFileName())
	if err != nil {
		log.Error().Err(err).Msg("can't delete")
		return
	}

	s.id = ""
	s.gg = nil
}
