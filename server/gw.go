package server

import (
	"errors"
	"fmt"

	"github.com/undeconstructed/gogogo/comms"
)

func encodeDown(down interface{}) (comms.Message, error) {
	switch msg := down.(type) {
	case comms.Message:
		// send preformatted message
		return msg, nil
	case responseToUser:
		// send response
		cmsg, err := comms.Encode("response:"+msg.ID, msg.Body)
		if err != nil {
			fmt.Printf("encode error: %v\n", err)
			break
		}
		return cmsg, nil
	case toSend:
		// send anything
		cmsg, err := comms.Encode(msg.mtype, msg.data)
		if err != nil {
			fmt.Printf("encode error: %v\n", err)
			return cmsg, nil
		}
		return cmsg, nil
	default:
		return comms.Message{}, fmt.Errorf("cannot send: %#v", msg)
	}

	return comms.Message{}, errors.New("huh")
}
