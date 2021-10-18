package comms

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

type ErrorCoder interface {
	ErrorCode() string
}

type CommsError struct {
	Code  string
	Cause error
}

func WrapError(err error) *CommsError {
	if err == nil {
		return nil
	}
	if ec, ok := err.(ErrorCoder); ok {
		return &CommsError{ec.ErrorCode(), err}
	}
	return &CommsError{"TODO", err}
}

func (e *CommsError) ErrorCode() string { return e.Code }
func (e *CommsError) String() string    { return e.Code + ": " + e.Error() }
func (e *CommsError) Error() string     { return e.Cause.Error() }

func (e *CommsError) MarshalJSON() ([]byte, error) {
	x := struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}{
		Code:    e.Code,
		Message: e.Cause.Error(),
	}
	return json.Marshal(x)
}

func (e *CommsError) UnmarshalJSON(b []byte) error {
	x := struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}{}
	if err := json.Unmarshal(b, &x); err != nil {
		return err
	}
	e.Code = x.Code
	if x.Message != "" {
		e.Cause = errors.New(x.Message)
	}
	return nil
}

type Head string

func headFromType(mtype string) Head {
	return Head(mtype)
}

func headFromFields(fields []string) Head {
	return Head(strings.Join(fields, ":"))
}

func headFromBytes(bytes []byte) Head {
	return Head(bytes)
}

func (h Head) Fields() []string {
	return strings.Split(string(h), ":")
}

func (h Head) Length() int {
	return len([]byte(h))
}

func (h Head) Bytes() []byte {
	return []byte(h)
}

type Message struct {
	Head Head
	Data []byte
}

func (m Message) Type() string {
	return m.Head.Fields()[0]
}

func Encode(mtype string, message interface{}) (Message, error) {
	var data []byte
	if mdata, ok := message.([]byte); ok {
		// already serial
		data = mdata
	} else {
		// encode to JSON
		jdata, err := json.Marshal(message)
		if err != nil {
			return Message{}, err
		}
		data = jdata
	}

	return Message{
		Head: headFromType(mtype),
		Data: data,
	}, nil
}

func Decode(m Message, v interface{}) error {
	err := json.Unmarshal(m.Data, v)
	if err != nil {
		return err
	}
	return nil
}

type Encoder struct {
	out io.Writer
}

func NewEncoder(out io.Writer) *Encoder {
	return &Encoder{
		out: out,
	}
}

func (enc *Encoder) Encode(mtype string, e interface{}) error {
	msg, err := Encode(mtype, e)
	if err != nil {
		return err
	}

	return enc.Send(msg)
}

func (enc *Encoder) Send(msg Message) error {
	headLength := uint32(msg.Head.Length())
	dataLength := uint32(len(msg.Data))
	length := headLength + dataLength + 12

	sizeBuf := make([]byte, 12)
	binary.BigEndian.PutUint32(sizeBuf, length)
	binary.BigEndian.PutUint32(sizeBuf[4:], headLength)
	binary.BigEndian.PutUint32(sizeBuf[8:], dataLength)

	_, err := enc.out.Write(sizeBuf)
	if err != nil {
		return err
	}
	_, err = enc.out.Write(msg.Head.Bytes())
	if err != nil {
		return err
	}
	_, err = enc.out.Write(msg.Data)
	if err != nil {
		return err
	}

	return nil
}

type Decoder struct {
	in io.Reader
}

func NewDecoder(in io.Reader) *Decoder {
	return &Decoder{
		in: in,
	}
}

func (dec *Decoder) Decode() (Message, error) {
	sizeBuf := make([]byte, 12)
	_, err := dec.in.Read(sizeBuf)
	if err != nil {
		return Message{}, err
	}

	length := binary.BigEndian.Uint32(sizeBuf)
	typeLength := binary.BigEndian.Uint32(sizeBuf[4:])
	dataLength := binary.BigEndian.Uint32(sizeBuf[8:])

	if length != typeLength+dataLength+12 {
		return Message{}, errors.New("bad data in pipe")
	}

	typeBuf := make([]byte, typeLength)
	_, err = dec.in.Read(typeBuf)
	if err != nil {
		return Message{}, err
	}

	dataBuf := make([]byte, dataLength)
	_, err = dec.in.Read(dataBuf)
	if err != nil {
		return Message{}, err
	}

	return Message{headFromBytes(typeBuf), dataBuf}, nil
}

type ConnectRequest struct {
	Msg string `json:"message"`
}

type ConnectResponse struct {
	Msg string      `json:"message"`
	Err *CommsError `json:"error"`
}
