package game

import "encoding/json"

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
			Custom: string(custom),
		})
	}

	return &RGameState{
		Status:  string(in.Status),
		Playing: in.Playing,
		Winner:  in.Winner,
		Players: players,
		Custom:  string(custom),
	}
}

func WrapTurnState(in *TurnState) *RTurnState {
	custom, _ := json.Marshal(in.Custom)

	return &RTurnState{
		Number: int32(in.Number),
		Player: in.Player,
		Can:    in.Can,
		Must:   in.Must,
		Custom: string(custom),
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
