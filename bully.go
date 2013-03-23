/*
 * Copyright 2013 Nan Deng
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/nu7hatch/gouuid"
	"math/big"
	"net"
	"strconv"
	"strings"
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
	ln       net.Listener
	myAddr   net.Addr
	myCAAddr string
}

type Candidate struct {
	Id   *big.Int
	Addr string
}

var ErrUnknownError = errors.New("Unknown")

func (self *Bully) MyId() *big.Int {
	return self.myId
}

func (self *Bully) MyAddr() string {
	if len(self.myCAAddr) == 0 {
		return fmt.Sprintf("localhost:%v", self.myport())
	}
	return self.myCAAddr
}

func (self *Bully) AddCandidate(addrStr string, id *big.Int, timeout time.Duration) error {
	addr, err := net.ResolveTCPAddr("tcp", addrStr)
	if err != nil {
		return err
	}
	if addr == nil {
		return ErrUnknownError
	}
	if timeout <= 1*time.Second {
		timeout = 2 * time.Second
	}
	ctrl := new(control)
	ctrl.addr = addrStr
	ctrl.id = id
	ctrl.cmd = ctrlADD
	ctrl.timeout = timeout
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

func (self *Bully) CandidateList() []*Candidate {
	ctrl := new(control)
	ctrl.cmd = ctrlQUERY_CANDY
	replyChan := make(chan *controlReply)
	ctrl.replyChan = replyChan

	self.ctrlChan <- ctrl

	ret := make([]*Candidate, 0, 10)
	for reply := range replyChan {
		c := new(Candidate)
		c.Id = reply.id
		c.Addr = reply.addr
		ret = append(ret, c)
	}
	return ret
}

func (self *Bully) Leader() (cand *Candidate, timestamp time.Time, err error) {
	ctrl := new(control)
	ctrl.cmd = ctrlQUERY_LEADER
	replyChan := make(chan *controlReply)
	ctrl.replyChan = replyChan

	self.ctrlChan <- ctrl
	reply := <-replyChan
	if reply == nil {
		err = ErrUnknownError
		return
	}
	if reply.err != nil {
		err = reply.err
		return
	}
	if len(reply.addr) == 0 || reply.id == nil {
		err = ErrUnknownError
		return
	}
	cand = new(Candidate)
	cand.Addr = reply.addr
	cand.Id = reply.id
	timestamp = reply.timestamp
	return
}

func (self *Bully) Finalize() {
	self.ln.Close()
	ctrl := new(control)
	ctrl.cmd = ctrlQUIT
	replyChan := make(chan *controlReply)
	ctrl.replyChan = replyChan

	self.ctrlChan <- ctrl
	<-replyChan
}

func (self *Bully) commandCollector(src *big.Int, conn net.Conn, cmdChan chan<- *command, timeout time.Duration) {
	defer conn.Close()
	for {
		cmd, err := readCommand(conn)
		if err != nil {
//			fmt.Printf("[COLLECTOR] I'm %v; error from %v: %v\n", self.myId, src, err)
			cmd := new(command)
			cmd.src = src
			cmd.Cmd = cmdBYE
			cmdChan <- cmd
			return
		}
		if cmd.Cmd == cmdITSME || cmd.Cmd == cmdBYE {
//			fmt.Printf("[COLLECTOR] I'm %v; message from %v: %v\n", self.myId, src, cmd.Cmd)
			cmd := new(command)
			cmd.src = src
			cmd.Cmd = cmdBYE
			cmdChan <- cmd
			return
		}
		if cmd.Cmd == cmdDUP_EXIT {
			return
		}
		cmd.src = src
		cmd.replyWriter = conn
		select {
		case cmdChan <- cmd:
			continue
		case <-time.After(timeout):
			cmd := new(command)
			cmd.src = src
			cmd.Cmd = cmdBYE
			cmdChan <- cmd
			return
		}
	}
}

func NewBully(ln net.Listener, myId *big.Int) *Bully {
	ret := new(Bully)
	if myId == nil {
		uu, _ := uuid.NewV4()
		ret.myId = stringToBig(uu.String())
	} else {
		ret.myId = myId
	}
	ret.cmdChan = make(chan *command)
	ret.ctrlChan = make(chan *control)
	ret.ln = ln
	go ret.listen(ln)
	go ret.process()
	return ret
}

type node struct {
	id     *big.Int
	conn   net.Conn
	caAddr string
}

func insertNode(l []*node, id *big.Int, conn net.Conn, caAddr string) ([]*node, bool) {
	n := findNode(l, id)
	if nil != n {
		return l, false
	}
	n = new(node)
	n.id = id
	n.conn = conn
	n.caAddr = caAddr
	return append(l, n), true
}

func dumpAllAddr(l []*node) []byte {
	addr := make([]string, 0, len(l))
	for _, n := range l {
		addr = append(addr, n.caAddr)
	}
	ret := strings.Join(addr, "\n")
	return []byte(ret)
}

func loadAllAddr(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	s := string(data)
	ret := strings.Split(s, "\n")
	return ret
}

func removeNode(l []*node, id *big.Int) []*node {
	idx := -1
	for i, n := range l {
		if n.id.Cmp(id) == 0 {
			idx = i
		}
	}
	if idx >= 0 {
		l[idx] = l[len(l)-1]
		l[len(l)-1] = nil
		l = l[:len(l)-1]
	}
	return l
}

func candyToString(l []*node) []string {
	ret := make([]string, len(l))
	for i, n := range l {
		ret[i] = n.caAddr
	}
	return ret
}

func findNodeByAddr(l []*node, addr string) *node {
	for _, n := range l {
		if n.caAddr == addr {
			return n
		}
	}
	return nil
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
	ctrlQUERY_LEADER
	ctrlQUERY_CANDY
	ctrlQUIT
)

type controlReply struct {
	addr string
	id   *big.Int
	timestamp time.Time
	err  error
}

type control struct {
	cmd       int
	addr      string
	id        *big.Int
	timeout   time.Duration
	replyChan chan<- *controlReply
}

var ErrUnmatchedId = errors.New("Unmatched node id")
var ErrTryLater = errors.New("Try it later")
var ErrBadProtoImpl = errors.New("Bad protocol implementation")

func (self *Bully) myport() int {
	addrStr := self.ln.Addr().String()
	ae := strings.Split(addrStr, ":")
	ret, _ := strconv.Atoi(ae[len(ae)-1])
	return ret
}

func (self *Bully) handshake(addr string, id *big.Int, candy []*node, timeout time.Duration) ([]*node, []string, error) {
//	fmt.Printf("[HANDSHAKE] I (%v) am shaking hands with %v\n", self.myId, addr)
	if addr == self.myCAAddr {
//		fmt.Printf("\tI (%v) am shaking hands with %v; It's me!\n", self.myId, addr)
		return candy, nil, nil
	}
	if findNodeByAddr(candy, addr) != nil {
//		fmt.Printf("\tI (%v) am shaking hands with %v; I have done this before!\n", self.myId, addr)
		return candy, nil, nil
	}
	if id != nil {
		cmp := id.Cmp(self.myId)
		if cmp > 0 {
			n := findNode(candy, id)
			if n != nil {
				return candy, nil, nil
			}
		} else if cmp < 0 {
			// If we know the id of a node,
			// then we only connect to the nodes with higher id,
			// and let the nodes with lower id connect us.
			return candy, nil, nil
		} else {
			// It is ourselves, don't need to add it.
			return candy, nil, nil
		}
	}
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return candy, nil, err
	}

	candyStrList := make([]string, 0, len(candy))
	for _, c := range candy {
		candyStrList = append(candyStrList, c.caAddr)
	}

	cmd := new(command)
	cmd.Cmd = cmdHELLO
	cmd.Header = make(map[string]string, 1)
	cmd.Header["port"] = fmt.Sprintf("%v", self.myport())
	cmd.Header["id"] = self.myId.String()
	cmd.Body = dumpAllAddr(candy)
//	fmt.Printf("[HANDSHAKE] I (%v) asked %v to shake hands with %v; %v\n", self.myId, addr, candyStrList, string(cmd.Body))
	err = writeCommand(conn, cmd)
	if err != nil {
		return candy, nil, err
	}
	reply, err := readCommand(conn)
	if err != nil {
		return candy, nil, err
	}
	if reply.Cmd != cmdHELLO_REPLY {
		switch reply.Cmd {
		case cmdTRY_LATER:
			conn.Close()
//			fmt.Printf("\tI (%v) am shaking hands with %v; he asked me to wait!\n", self.myId, addr)
			return candy, nil, ErrTryLater
		case cmdDUP_CONN:
//			fmt.Printf("\tI (%v) am shaking hands with %v; DUPCONN\n", self.myId, addr)
			reply := new(command)
			reply.Cmd = cmdDUP_EXIT
			writeCommand(conn, reply)
			conn.Close()
			return candy, nil, nil
		case cmdITSME:
//			fmt.Printf("\tI (%v) am shaking hands with %v; ME\n", self.myId, addr)
			self.myAddr, _ = net.ResolveTCPAddr("tcp", addr)
			self.myCAAddr = addr
			conn.Close()
			return candy, nil, nil
		}
		return candy, nil, ErrBadProtoImpl
	}
	if len(reply.Header) == 0 {
		return candy, nil, ErrBadProtoImpl
	}
	if _, ok := reply.Header["id"]; !ok {
		return candy, nil, ErrBadProtoImpl
	}

	rId := new(big.Int)
	rId, _ = rId.SetString(reply.Header["id"], 10)
	if rId == nil {
		return candy, nil, ErrBadProtoImpl
	}
	if id != nil {
		if rId.Cmp(id) != 0 {
			return candy, nil, ErrUnmatchedId
		}
	}
	candyList := loadAllAddr(reply.Body)
	moreCandy := make([]string, 0, len(candyList))
	for _, c := range candyList {
		if c == self.myCAAddr {
			continue
		}
		n := findNodeByAddr(candy, c)
		if n == nil {
			moreCandy = append(moreCandy, c)
		}
	}

//	fmt.Printf("[HANDSHAKE] I (%v) have shaked hand with %v (%v), he asked me to shake hands with %v\n", self.myId, addr, rId, moreCandy)

	candy, _ = insertNode(candy, rId, conn, addr)
	go self.commandCollector(rId, conn, self.cmdChan, 10*time.Second)

	return candy, moreCandy, nil
}

func (self *Bully) electUntilDie(candy []*node, timeout time.Duration) ([]*node, *node) {
//	fmt.Printf("[ELECT] I (%v) (%v) initiated an election\n", self.myId, self.myport())
	leader, candy, err := self.elect(candy, timeout)
	// Try until we get a leader
	for leader == nil || err != nil {
		leader, candy, err = self.elect(candy, timeout)
	}
	return candy, leader
}

var ErrNeedNewElection = errors.New("Need another round")

func (self *Bully) elect(candy []*node, timeout time.Duration) (leader *node, newCandy []*node, err error) {
	higherCandy := make([]*node, 0, len(candy))
//	fmt.Printf("[ELECT TMP] my (%v) candy list %v\n", self.myId, candyToString(candy))
	newCandy = candy
	for _, c := range candy {
		if c.id.Cmp(self.myId) > 0 {
//			fmt.Printf("[ELECT TMP] I (%v) send elect message to %v\n", self.myId, c.id)
			cmd := new(command)
			cmd.Cmd = cmdELECT
			err := writeCommand(c.conn, cmd)
			if err == nil {
				higherCandy = append(higherCandy, c)
			}
		}
	}

	// No one is higher than me.
	// I am the leader.
	if len(higherCandy) <= 0 {
//		fmt.Printf("[ELECT RESULT] I (%v) am the leader\n", self.myId)
		leader = new(node)
		leader.conn = nil
		leader.id = self.myId
		leader.caAddr = self.myCAAddr
		for _, c := range candy {
			cmd := new(command)
			cmd.Cmd = cmdCOORDIN
			writeCommand(c.conn, cmd)
		}
		return
	}
	slaved := false
	for {
		select {
		case cmd := <-self.cmdChan:
			switch cmd.Cmd {
			case cmdBYE:
//				fmt.Printf("[BYEINELECTION] I (%v) received bye bye from %v \n", self.myId, cmd.src)
				candy = removeNode(candy, cmd.src)
				newCandy = candy
				err = ErrNeedNewElection
				return
			case cmdHELLO:
				reply := new(command)
				reply.Cmd = cmdTRY_LATER
				writeCommand(cmd.replyWriter, reply)
			case cmdELECT:
				reply := new(command)
				reply.Cmd = cmdELECT_OK
				writeCommand(cmd.replyWriter, reply)
			case cmdELECT_OK:
//				fmt.Printf("[ELECT TMP] I (%v) received reply from %v\n", self.myId, cmd.src)
				n := findNode(higherCandy, cmd.src)
				if n == nil {
					continue
				}
				slaved = true
			case cmdCOORDIN:
				n := findNode(candy, cmd.src)
//				fmt.Printf("[ELECT RESULT] %v is the leader\n", cmd.src)
				if n == nil {
					err = ErrNeedNewElection
					return
				}
				if n.id.Cmp(self.myId) < 0 {
					err = ErrNeedNewElection
					return
				}
				leader = n
				return
			}
		case <-time.After(timeout):
			break
		}
	}

	// No one replied within time out.
	// I am the leader.
	if !slaved {
		leader = new(node)
		leader.conn = nil
		leader.id = self.myId
		leader.caAddr = self.myCAAddr
//		fmt.Printf("[ELECT RESULT] I (%v) am the leader\n", self.myId)
		for _, c := range candy {
			cmd := new(command)
			cmd.Cmd = cmdCOORDIN
			writeCommand(c.conn, cmd)
		}
		return
	}
	err = ErrNeedNewElection
	return
}

func getIp(addr string) string {
	ae := strings.Split(addr, ":")
	if len(ae) == 0 {
		return ""
	}
	return strings.Join(ae[:len(ae)-1], ":")
}

func (self *Bully) localhost() net.Addr {
	addrStr := self.ln.Addr().String()
	ae := strings.Split(addrStr, ":")
	addrStr = fmt.Sprintf("127.0.0.1:%v", ae[len(ae)-1])
	ret, _ := net.ResolveTCPAddr("tcp", addrStr)
	return ret
}

func (self *Bully) addCandidates(candy []*node, candyList []string, timeout time.Duration) (ret []*node, err error) {
	newCandies := make([]string, 0, 10)
	for len(candyList) > 0 {
		for _, c := range candyList {
			if findNodeByAddr(candy, c) != nil {
				continue
			}
			var l []string
			candy, l, err = self.handshake(c, nil, candy, timeout)
			for _, i := range l {
				found := false
				for _, j := range newCandies {
					if i == j {
						found = true
					}
				}
				if !found {
					newCandies = append(newCandies, i)
				}
			}
			if err == ErrTryLater {
				newCandies = append(newCandies, c)
			}
		}
		candyList = newCandies
		newCandies = candyList[:0]
	}
	ret = candy
	return
}

func (self *Bully) process() {
	candy := make([]*node, 0, 10)
	var leader *node
	leaderTimeout := 10 * time.Second
	var leaderTimeStamp time.Time
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
					caAddr := ""
					if port, ok := cmd.Header["port"]; ok {
						caAddr = fmt.Sprintf("%v:%v", getIp(cmd.replyWriter.RemoteAddr().String()), port)
					}
					reply := new(command)
					if len(caAddr) == 0 {
						reply.Cmd = cmdBYE
						writeCommand(cmd.replyWriter, reply)
						continue
					}
					candyList := loadAllAddr(cmd.Body)
//					fmt.Printf("[CANDY] I (%v) received handshake from %v (%v), he asked me to shake hands with %v; %v\n", self.myId, caAddr, cmd.src, candyList, string(cmd.Body))

					candy, _ = insertNode(candy, cmd.src, cmd.replyWriter, caAddr)
					candy, _ = self.addCandidates(candy, candyList, leaderTimeout)
//					fmt.Printf("[CANDYLIST] I (%v) have candies: %v\n", self.myId, candyToString(candy))

					reply.Cmd = cmdHELLO_REPLY
					reply.Header = make(map[string]string, 1)
					reply.Header["id"] = self.myId.String()
					reply.Body = dumpAllAddr(candy)
//					fmt.Printf("[HS] I (%v) asked %v (%v) to shake hands with %v\n", self.myId, caAddr, cmd.src, candyToString(candy))
					err := writeCommand(cmd.replyWriter, reply)
					if err != nil {
						cmd.replyWriter.Close()
						continue
					}
				} else {
					reply := new(command)
					reply.Cmd = cmdDUP_CONN
					writeCommand(cmd.replyWriter, reply)
				}
			case cmdBYE:
//				fmt.Printf("[BYE] I (%v) received bye bye from %v \n", self.myId, cmd.src)
				candy = removeNode(candy, cmd.src)
				if leader == nil || leader.id == nil || cmd.src.Cmp(leader.id) == 0 {
					candy, leader = self.electUntilDie(candy, leaderTimeout)
					leaderTimeStamp = time.Now()
				}
			case cmdELECT:
				reply := new(command)
				reply.Cmd = cmdELECT_OK
				err := writeCommand(cmd.replyWriter, reply)
				if err != nil {
					continue
				}
//				fmt.Printf("[ELECT] I (%v) received election from %v \n", self.myId, cmd.src)
				candy, leader = self.electUntilDie(candy, leaderTimeout)
				leaderTimeStamp = time.Now()
//				fmt.Printf("[ELECT] I (%v) received election from %v; and the leader is %v(%v) \n", self.myId, cmd.src, leader.caAddr, leader.id)
			case cmdCOORDIN:
				n := findNode(candy, cmd.src)
//				fmt.Printf("[ELECT-RESULT] I (%v) received election result from %v and he is the leader\n", self.myId, cmd.src)
				if n == nil {
					candy, leader = self.electUntilDie(candy, leaderTimeout)
					leaderTimeStamp = time.Now()
				} else if n.id.Cmp(self.myId) < 0 {
					candy, leader = self.electUntilDie(candy, leaderTimeout)
					leaderTimeStamp = time.Now()
				} else {
					leader = n
				}
			}
		case ctrl := <-self.ctrlChan:
			switch ctrl.cmd {
			case ctrlADD:
				var err error
				oldCandyLen := len(candy)
				candyList := make([]string, 1, 10)
				candyList[0] = ctrl.addr
//				fmt.Printf("[ADDBEFORE] I (%v) was aksed to add %v. Now my candy list contains: %v\n", self.myId, ctrl.addr, candyToString(candy))
				candy, err = self.addCandidates(candy, candyList, ctrl.timeout)
				reply := new(controlReply)
				if err != nil {
					reply.err = err
					ctrl.replyChan <- reply
//					fmt.Printf("[ADDABORT] I (%v) was aksed to add %v. Now it's wrong: %v\n", self.myId, ctrl.addr, err)
					continue
				}
				if len(candy) > oldCandyLen || leader == nil {
//					fmt.Printf("[NEED ELECTION] I (%v) have added %v. Now my candy list contains: %v\n", self.myId, ctrl.addr, candyToString(candy))
					candy, leader = self.electUntilDie(candy, leaderTimeout)
					leaderTimeStamp = time.Now()
//					fmt.Printf("[ELECT] I (%v) initiated the election and the leader is: %v(%v) \n", self.myId, leader.caAddr, leader.id)
				}
//				fmt.Printf("[ADDAFTER] I (%v) have added %v. Now my candy list contains: %v; and the leader is %v(%v)\n", self.myId, ctrl.addr, candyToString(candy), leader.caAddr, leader.id)
				ctrl.replyChan <- reply
			case ctrlQUERY_CANDY:
				reply := new(controlReply)
				reply.addr = self.MyAddr()
				reply.id = self.myId
				ctrl.replyChan <- reply
				for _, node := range candy {
					reply := new(controlReply)
					reply.addr = node.caAddr
					reply.id = node.id
					ctrl.replyChan <- reply
				}
				close(ctrl.replyChan)
			case ctrlQUERY_LEADER:
				if leader == nil {
//					fmt.Printf("[QUERY SO NEED ELECTION] My (%v) candy list contains: %v\n", self.myId, candyToString(candy))
					candy, leader = self.electUntilDie(candy, leaderTimeout)
					leaderTimeStamp = time.Now()
				}
				reply := new(controlReply)
				if leader.conn != nil {
					reply.addr = leader.caAddr
				} else {
					if len(self.myCAAddr) != 0 {
						reply.addr = self.myCAAddr
					} else {
						reply.addr = "localhost"
					}
				}
				reply.timestamp = leaderTimeStamp
				reply.id = leader.id
				ctrl.replyChan <- reply
			case ctrlQUIT:
				for _, node := range candy {
					reply := new(command)
					reply.Cmd = cmdBYE
					writeCommand(node.conn, reply)
				}

				// Leader should wait for the replies
				if len(candy) > 0 {
					for cmd := range self.cmdChan {
						if cmd.Cmd == cmdBYE {
//							fmt.Printf("[BYEWAIT] I (%v) received bye bye from %v \n", self.myId, cmd.src)
							candy = removeNode(candy, cmd.src)
						}
						if len(candy) == 0 {
							break
						}
					}
				}
//				fmt.Printf("[BYEDONE] I (%v) will never take a little piece of cloud\n88888888888888\n", self.myId)
				close(ctrl.replyChan)
				return
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
	if len(cmd.Header) != 2 {
		conn.Close()
		return
	}

	idStr, ok := cmd.Header["id"]
	if !ok {
		conn.Close()
		return
	}
	rId, _ := new(big.Int).SetString(idStr, 10)
	if rId == nil {
		conn.Close()
		return
	}
	if rId.Cmp(self.myId) == 0 {
		reply := new(command)
		reply.Cmd = cmdITSME
		writeCommand(conn, reply)
		conn.Close()
		return
	}
//	fmt.Printf("[ANOTHER] I (%v) received handshake from %v, his candy list %v\n", self.myId, idStr, string(cmd.Body))
	cmd.src = rId
	cmd.replyWriter = conn
	go self.commandCollector(rId, conn, self.cmdChan, 10*time.Second)
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
			return
		}
		if conn == nil {
			continue
		}
		go self.replyHandshake(conn)
	}
}
