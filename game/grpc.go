package game

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func WrapChanges(in []Change) []*RChange {
	var out []*RChange
	for _, c := range in {
		out = append(out, &RChange{
			Who:   c.Who,
			What:  c.What,
			Where: c.Where,
		})
	}

	return out
}

func WrapGameState(in *GameState) *RGameState {
	custom, _ := json.Marshal(in.Custom)

	var players []*RPlayer
	for _, p := range in.Players {
		custom, _ := json.Marshal(p.Custom)
		players = append(players, &RPlayer{
			Name:   p.Name,
			Colour: p.Colour,
			Custom: custom,
		})
	}

	return &RGameState{
		Status:  string(in.Status),
		Playing: in.Playing,
		Winner:  in.Winner,
		Players: players,
		Custom:  custom,
	}
}

func WrapTurnState(in *TurnState) *RTurnState {
	custom, _ := json.Marshal(in.Custom)

	return &RTurnState{
		Number: int32(in.Number),
		Player: in.Player,
		Can:    in.Can,
		Must:   in.Must,
		Custom: custom,
	}
}

func UnwrapChanges(in []*RChange) []Change {
	var out []Change
	for _, c := range in {
		out = append(out, Change{
			Who:   c.Who,
			What:  c.What,
			Where: c.Where,
		})
	}

	return out
}

func UnwrapGameState(in *RGameState) *GameState {
	var players []PlayerState
	for _, p := range in.Players {
		players = append(players, PlayerState{
			Name:   p.Name,
			Colour: p.Colour,
			Custom: json.RawMessage(p.Custom),
		})
	}

	return &GameState{
		Status:  GameStatus(in.Status),
		Playing: in.Playing,
		Winner:  in.Winner,
		Players: players,
		Custom:  json.RawMessage(in.Custom),
	}
}

func UnwrapTurnState(in *RTurnState) *TurnState {
	return &TurnState{
		Number: int(in.Number),
		Player: in.Player,
		Can:    in.Can,
		Must:   in.Must,
		Custom: json.RawMessage(in.Custom),
	}
}

type NewGameFunc func(map[string]interface{}) (Game, error)
type LoadGameFunc func(io.Reader) (Game, error)

func GRPCMain(newGame NewGameFunc, loadGame LoadGameFunc) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	bind := os.Args[1]

	gsrv, err := NewGRPCServer(bind, newGame, loadGame)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		panic("cannot make gsrv")
	}

	rand.Seed(time.Now().Unix())

	err = gsrv.StartServer(context.Background())
	if err != nil {
		panic("error")
	}
}

type GRPCServer struct {
	UnimplementedInstanceServer

	newGame  NewGameFunc
	loadGame LoadGameFunc

	listener net.Listener

	id string
	gg Game
}

func NewGRPCServer(bind string, newGame NewGameFunc, loadGame LoadGameFunc) (*GRPCServer, error) {
	l, err := net.Listen("unix", bind)
	if err != nil {
		return nil, err
	}
	return &GRPCServer{
		newGame:  newGame,
		loadGame: loadGame,
		listener: l,
	}, nil
}

func (s *GRPCServer) StartServer(ctx context.Context) error {
	srv := grpc.NewServer()
	RegisterInstanceServer(srv, s)

	return srv.Serve(s.listener)
}

func (s *GRPCServer) Load(ctx context.Context, req *RLoadRequest) (*RLoadResponse, error) {
	if s.gg != nil {
		return nil, status.Errorf(codes.AlreadyExists, "game already present")
	}

	f, err := os.Open("state-" + req.Id + ".json")
	if err != nil {
		log.Error().Err(err).Msg("cannot open state file")
		return nil, err
	}

	gg, err := s.loadGame(f)
	if err != nil {
		log.Error().Err(err).Msg("cannot restore state")
		return nil, err
	}

	s.id = req.Id
	s.gg = gg

	sg := s.gg.GetGameState()
	st := s.gg.GetTurnState()

	return &RLoadResponse{
		State: WrapGameState(&sg),
		Turn:  WrapTurnState(&st),
	}, nil
}

func (s *GRPCServer) Init(ctx context.Context, req *RInitRequest) (*RInitResponse, error) {
	if s.gg != nil {
		return nil, status.Errorf(codes.AlreadyExists, "game already present")
	}

	options := map[string]interface{}{}
	err := json.Unmarshal(req.Options, &options)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad options json")
	}

	s.id = req.Id
	gg, _ := s.newGame(options)
	s.gg = gg

	err = s.saveGame()
	if err != nil {
		// XXX - this and others are dangerous, as they break sync
		log.Error().Err(err).Msg("save failed")
	}

	sg := s.gg.GetGameState()

	return &RInitResponse{
		State: WrapGameState(&sg),
	}, nil
}

func (s *GRPCServer) AddPlayer(ctx context.Context, in *RAddPlayerRequest) (*RAddPlayerResponse, error) {
	if s.gg == nil {
		panic("no game")
	}

	err := s.gg.AddPlayer(in.Name, in.Colour)
	if err != nil {
		switch Code(err) {
		case StatusBadRequest:
			return nil, status.Errorf(codes.InvalidArgument, "%v", err)
		default:
			return nil, status.Errorf(codes.FailedPrecondition, "%v", err)
		}
	}
	err = s.saveGame()
	if err != nil {
		log.Error().Err(err).Msg("save failed")
	}

	sg := s.gg.GetGameState()

	return &RAddPlayerResponse{
		State: WrapGameState(&sg),
	}, nil
}

func (s *GRPCServer) Start(context.Context, *RStartRequest) (*RStartResponse, error) {
	if s.gg == nil {
		panic("no game")
	}

	st, err := s.gg.Start()
	if err != nil {
		switch Code(err) {
		case StatusBadRequest:
			return nil, status.Errorf(codes.InvalidArgument, "%v", err)
		default:
			return nil, status.Errorf(codes.Unknown, "%v", err)
		}
	}
	err = s.saveGame()
	if err != nil {
		log.Error().Err(err).Msg("save failed")
	}

	sg := s.gg.GetGameState()

	return &RStartResponse{
		State: WrapGameState(&sg),
		Turn:  WrapTurnState(&st),
	}, nil
}

func (s *GRPCServer) Play(ctx context.Context, in *RPlayRequest) (*RPlayResponse, error) {
	if s.gg == nil {
		panic("no game")
	}

	res, err := s.gg.Play(in.Player, Command{
		Command: CommandString(in.Command),
		Options: in.Options,
	})
	if err != nil {
		switch Code(err) {
		case StatusNotStarted, StatusNotYourTurn, StatusNotNow, StatusMustDo, StatusWrongPhase:
			return nil, status.Errorf(codes.FailedPrecondition, "%v", err)
		case StatusBadRequest:
			return nil, status.Errorf(codes.InvalidArgument, "%v", err)
		default:
			return nil, status.Errorf(codes.Unknown, "%v", err)
		}
	}
	err = s.saveGame()
	if err != nil {
		log.Error().Err(err).Msg("save failed")
	}

	rr, _ := json.Marshal(res.Response)

	sg := s.gg.GetGameState()

	return &RPlayResponse{
		Response: rr,
		News:     WrapChanges(res.News),
		State:    WrapGameState(&sg),
		Turn:     WrapTurnState(&res.Next),
	}, nil
}

func (s *GRPCServer) Destroy(context.Context, *RDestroyRequest) (*RDestroyResponse, error) {
	if s.gg == nil {
		panic("no game")
	}

	err := s.wipeGame()
	if err != nil {
		log.Error().Err(err).Msg("cannot delete")
		return nil, status.Error(codes.Internal, "cannot delete")
	}
	s.gg = nil

	return &RDestroyResponse{}, nil
}

func (s *GRPCServer) saveFileName() string {
	return fmt.Sprintf("state-%s.json", s.id)
}

func (s *GRPCServer) saveGame() error {
	if s.gg == nil {
		panic("no game")
	}

	outFile, err := os.Create(s.saveFileName())
	if err != nil {
		return err
	}
	defer outFile.Close()

	return s.gg.WriteOut(outFile)
}

func (s *GRPCServer) wipeGame() error {
	if s.gg == nil {
		panic("no game")
	}

	err := os.Remove(s.saveFileName())
	if err != nil {
		return err
	}

	s.id = ""
	s.gg = nil

	return nil
}
