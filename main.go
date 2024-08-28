package main

import (
	"log/slog"
	"net"
)

const (
	defaultListenAddr = ":5001"
)

type Config struct {
	ListenAddr string
}

type Service struct {
	Config
	peers      map[*Peer]bool
	ln         net.Listener
	addPeerCh  chan *Peer
	quitPeerCh chan struct{}
	msgCh      chan []byte
}

func NewService(cfg Config) *Service {
	if len(cfg.ListenAddr) == 0 {
		cfg.ListenAddr = defaultListenAddr
	}
	return &Service{
		Config:     cfg,
		peers:      make(map[*Peer]bool),
		addPeerCh:  make(chan *Peer),
		quitPeerCh: make(chan struct{}),
	}
}

func (s *Service) Start() error {
	listen, err := net.Listen("tcp", s.ListenAddr)
	if err != nil {
		return err
	}
	s.ln = listen
	go s.loop()
	slog.Info("service running", "start", s.ListenAddr)
	return s.acceptLoop()
}

func (s *Service) loop() {
	for {
		select {
		case peer := <-s.addPeerCh:
			s.peers[peer] = true
		case <-s.quitPeerCh:
			return
		// 接收到消息
		case <-s.msgCh:

		default:

		}
	}
}

func (s *Service) acceptLoop() error {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			slog.Error("accept error", "err", err)
			continue
		}
		go s.handleConn(conn)
	}
}

func (s *Service) handleConn(conn net.Conn) {
	peer := NewPeer(conn, s.msgCh)
	s.addPeerCh <- peer
	slog.Info("new peer connected", "remoteAddr", conn.RemoteAddr())
	go func() {
		err := peer.readLoop()
		if err != nil {
			slog.Error("readLoop error", "err", err)
		}
	}()
}

func main() {
	NewService(Config{}).Start()
}
