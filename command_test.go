package main

import (
	"testing"
	"bytes"
	"math/big"
)

func cmdEq(a, b *command) bool {
	if a.Cmd != b.Cmd {
		return false
	}
	if len(a.Header) != len(b.Header) {
		return false
	}
	for k, v := range a.Header {
		if b.Header[k] != v {
			return false
		}
	}
	for i, x := range a.Body {
		if b.Body[i] != x {
			return false
		}
	}
	return true
}

func testWriteThenRead(cmd *command, t *testing.T) {
	buf := make([]byte, 0, 2048)
	byteBuffer := bytes.NewBuffer(buf)
	err := writeCommand(byteBuffer, cmd)
	if err != nil {
		t.Errorf("Write error: %v\n", err)
	}

	shadow, err := readCommand(byteBuffer)
	if err != nil {
		t.Errorf("Read error: %v\n", err)
	}

	cmd.src, _ = new(big.Int).SetString("123456", 10)
	if !cmdEq(shadow, cmd) {
		t.Errorf("Not equal")
	}
}

func TestFullCommand(t *testing.T) {
	cmd := new(command)
	cmd.Cmd = uint8(1)
	cmd.Header = make(map[string]string, 10)
	cmd.Header["h1"] = "header 1"
	cmd.Header["h2"] = "header 2"
	cmd.Body = []byte{1,2,54,2}
	testWriteThenRead(cmd, t)
}

func TestCommandWithBodyOnly(t *testing.T) {
	cmd := new(command)
	cmd.Cmd = uint8(1)
	cmd.Header = make(map[string]string, 10)
	cmd.Body = []byte{1,2,54,2}
	testWriteThenRead(cmd, t)
}

func TestCommandWithHeaderOnly(t *testing.T) {
	cmd := new(command)
	cmd.Cmd = uint8(1)
	cmd.Header = make(map[string]string, 10)
	cmd.Header["h1"] = "header 1"
	cmd.Header["h2"] = "header 2"
	testWriteThenRead(cmd, t)
}

