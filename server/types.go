package server

type toSend struct {
	mtype string
	data  interface{}
}

type ConnectMsg struct {
	Name   string
	Colour string
	Client clientBundle
	Rep    chan error
}

type DisconnectMsg struct {
	Name string
}

type TextFromUser struct {
	Who  string
	Text string
}

type RequestFromUser struct {
	Who  string
	ID   string
	Cmd  []string
	Body interface{}
}

type ResponseToUser struct {
	ID   string
	Body interface{}
}

type clientBundle struct {
	downCh chan interface{}
}
