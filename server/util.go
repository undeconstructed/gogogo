package server

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"strings"
)

var letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func encodeConnectString(gameId, playerId string) string {
	s := fmt.Sprintf("%s//%s", gameId, playerId)
	c := base64.StdEncoding.EncodeToString([]byte(s))
	return c
}

func decodeConnectString(code string) (gameId, playerId string, err error) {
	s, err := base64.StdEncoding.DecodeString(code)
	if err != nil {
		return "", "", errors.New("bad code")
	}
	ss := strings.Split(string(s), "//")
	if len(ss) != 2 {
		return "", "", errors.New("bad code")
	}
	return ss[0], ss[1], nil
}
