package main

import (
	"net"
)

const (
	CommentSet = "SET"
)

type Peer struct {
	conn  net.Conn
	msgCh chan []byte
}

func NewPeer(conn net.Conn, msg chan []byte) *Peer {
	return &Peer{conn: conn,
		msgCh: msg,
	}
}

func (p *Peer) readLoop() error {
	buf := make([]byte, 1024)
	for {
		n, err := p.conn.Read(buf)
		if err != nil {
			return err
		}
		//fmt.Println("readLoop", string(buf[:n]))
		msgBuf := make([]byte, n)
		copy(msgBuf, buf[:n])
		p.msgCh <- msgBuf
	}
}

func (p *Peer) handleMSG(msg []byte) {

}
