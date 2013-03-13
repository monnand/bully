package main

import (
	"crypto/sha256"
	"errors"
	"github.com/nu7hatch/gouuid"
	"math/big"
	"net"
	"time"
)

func stringToBig(str string) *big.Int {
	sha := sha256.New()
	sha.Write([]byte(str))
	b := sha.Sum(nil)

	ret := new(big.Int).SetBytes(b)
	return ret
}

type Bully struct {
	myId     *big.Int
	cmdChan  chan *command
	ctrlChan chan *control
}

var ErrUnknownError = errors.New("Unknown")

func (self *Bully) AddCandidate(addrStr string, id *big.Int) error {
	addr, err := net.ResolveTCPAddr("tcp", addrStr)
	if err != nil {
		return err
	}
	ctrl := new(control)
	ctrl.addr = addr
	ctrl.id = id
	ctrl.cmd = ctrlADD
	replyChan := make(chan *controlReply)
	ctrl.replyChan = replyChan

	self.ctrlChan <- ctrl
	reply := <-replyChan
	if reply == nil {
		return ErrUnknownError
	}
	if reply.err != nil {
		return reply.err
	}
	return nil
}

func commandCollector(src *big.Int, conn net.Conn, cmdChan chan<- *command, timeout time.Duration) {
	defer conn.Close()
	for {
		cmd, err := readCommand(conn)
		if err != nil {
			return
		}
		if cmd.Cmd == cmdITSME || cmd.Cmd == cmdBYE {
			return
		}
		cmd.src = src
		cmd.replyWriter = conn
		select {
		case cmdChan <- cmd:
			continue
		case <-time.After(timeout):
			return
		}
	}
}

func NewBully(ln net.Listener, myId *big.Int) *Bully {
	ret := new(Bully)
	if myId != nil {
		uu, _ := uuid.NewV4()
		ret.myId = stringToBig(uu.String())
	} else {
		ret.myId = myId
	}
	ret.cmdChan = make(chan *command)
	ret.ctrlChan = make(chan *control)
	go ret.listen(ln)
	go ret.process()
	return ret
}

type node struct {
	id   *big.Int
	conn net.Conn
}

func insertNode(l []*node, id *big.Int, conn net.Conn) ([]*node, bool) {
	n := findNode(l, id)
	if nil != n {
		return l, false
	}
	n = new(node)
	n.id = id
	n.conn = conn
	return append(l, n), true
}

func findNode(l []*node, id *big.Int) *node {
	for _, n := range l {
		if n.id.Cmp(id) == 0 {
			return n
		}
	}
	return nil
}

const (
	ctrlADD = iota
	ctrlQUERY
)

type controlReply struct {
	addr net.Addr
	id   *big.Int
	err  error
}

type control struct {
	cmd       int
	addr      net.Addr
	id        *big.Int
	replyChan chan<- *controlReply
}

var ErrUnmatchedId = errors.New("Unmatched node id")

func (self *Bully) handshake(addr net.Addr, id *big.Int, candy []*node) error {
	if id != nil {
		cmp := id.Cmp(self.myId)
		if cmp > 0 {
			n := findNode(candy, id)
			if n != nil {
				return nil
			}
		} else if cmp < 0 {
			// If we know the id of a node,
			// then we only connect to the nodes with higher id,
			// and let the nodes with lower id connect us.
			return nil
		} else {
			// It is ourselves, don't need to add it.
			return nil
		}
	}
	conn, err := net.Dial("tcp", addr.String())
	if err != nil {
		return err
	}
	cmd := new(command)
	cmd.Cmd = cmdHELLO
	cmd.Body = self.myId.Bytes()
	err = writeCommand(conn, cmd)
	if err != nil {
		return err
	}
	reply, err := readCommand(conn)
	if err != nil {
		return err
	}
	if reply.Cmd != cmdHELLO_REPLY {
		conn.Close()
		return nil
	}
	rId := new(big.Int).SetBytes(reply.Body)
	if id != nil {
		if rId.Cmp(id) != 0 {
			return ErrUnmatchedId
		}
	}
	candy, _ = insertNode(candy, rId, conn)
	go commandCollector(rId, conn, self.cmdChan, 10*time.Second)

	return nil
}

func (self *Bully) process() {
	candy := make([]*node, 0, 10)
	//var leader *node
	for {
		select {
		case cmd := <-self.cmdChan:
			switch cmd.Cmd {
			case cmdHELLO:
				if cmd.src.Cmp(self.myId) == 0 {
					reply := new(command)
					reply.Cmd = cmdITSME
					err := writeCommand(cmd.replyWriter, reply)
					if err != nil {
						cmd.replyWriter.Close()
					}
					continue
				}
				n := findNode(candy, cmd.src)
				if n == nil {
					reply := new(command)
					reply.Cmd = cmdHELLO_REPLY
					reply.Body = self.myId.Bytes()
					err := writeCommand(cmd.replyWriter, reply)
					if err != nil {
						cmd.replyWriter.Close()
						continue
					}
					candy, _ = insertNode(candy, cmd.src, cmd.replyWriter)
				} else {
					reply := new(command)
					reply.Cmd = cmdDUP_CONN
					writeCommand(cmd.replyWriter, reply)
					cmd.replyWriter.Close()
				}
			}
		case ctrl := <-self.ctrlChan:
			switch ctrl.cmd {
			case ctrlADD:
				err := self.handshake(ctrl.addr, ctrl.id, candy)
				reply := new(controlReply)
				if err != nil {
					reply.err = err
					ctrl.replyChan <- reply
					continue
				}
				ctrl.replyChan <- reply
			}
		}
	}
}

func (self *Bully) replyHandshake(conn net.Conn) {
	cmd, err := readCommand(conn)
	if err != nil {
		conn.Close()
		return
	}
	if cmd.Cmd != cmdHELLO {
		conn.Close()
		return
	}
	if len(cmd.Body) == 0 {
		conn.Close()
		return
	}

	rId := new(big.Int).SetBytes(cmd.Body)
	cmd.src = rId
	cmd.replyWriter = conn
	go commandCollector(rId, conn, self.cmdChan, 10*time.Second)
	select {
	case self.cmdChan <- cmd:
	case <-time.After(10 * time.Second):
	}
	return
}

func (self *Bully) listen(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go self.replyHandshake(conn)
	}
}
