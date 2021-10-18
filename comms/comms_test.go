package comms

import (
	"bytes"
	"testing"
)

func TestEncDec(t *testing.T) {
	var network bytes.Buffer
	enc := NewEncoder(&network)
	dec := NewDecoder(&network)

	err := enc.Encode("test", "data")
	if err != nil {
		t.Errorf("enc error: %v", err)
	}

	msg, err := dec.Decode()
	if t0 := msg.Type(); t0 != "test" {
		t.Errorf("bad decode: %v", t0)
	}
	if string(msg.Data) != "\"data\"" {
		t.Errorf("bad decode: %v", msg.Data)
	}
}
