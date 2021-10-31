package server

type toSend struct {
	mtype string
	data  interface{}
}

type createGameMsg struct {
	Name string
	Rep  chan error
}

type connectMsg struct {
	Game   string
	Name   string
	Colour string
	Client clientBundle
	Rep    chan error
}

type disconnectMsg struct {
	Game string
	Name string
}

type textFromUser struct {
	Game string
	Who  string
	Text string
}

type requestFromUser struct {
	Game string
	Who  string
	ID   string
	Cmd  []string
	Body interface{}
}

type responseToUser struct {
	ID   string
	Body interface{}
}

type clientBundle struct {
	downCh chan interface{}
}
