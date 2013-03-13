package main

import (
	"labix.org/v2/mgo/bson"
	"io"
	"encoding/binary"
	"errors"
)

type command struct {
	Cmd uint8
	Header map[string]string ",omitempty"
	Body []byte ",omitempty"
}

var ErrCannotReadFull = errors.New("Cannot read full length")

func readCommand(reader io.Reader) (cmd *command, err error) {
	var cmdLen uint16
	err = binary.Read(reader, binary.BigEndian, &cmdLen)
	if err != nil {
		return
	}
	data := make([]byte, int(cmdLen))
	n, err := io.ReadFull(reader, data)
	if err != nil {
		return
	}
	if n != len(data) {
		err = ErrCannotReadFull
		return
	}
	cmd = new(command)
	err = bson.Unmarshal(data, cmd)
	return
}

func writen(w io.Writer, buf []byte) error {
	n := len(buf)
	for n >= 0 {
		l, err := w.Write(buf)
		if err != nil {
			return err
		}
		if l >= n {
			return nil
		}
		n -= l
		buf = buf[l:]
	}
	return nil
}

func writeCommand(writer io.Writer, cmd *command) error {
	var cmdLen uint16
	data, err := bson.Marshal(cmd)
	if err != nil {
		return err
	}
	cmdLen = uint16(len(data))
	err = binary.Write(writer, binary.BigEndian, cmdLen)
	if err != nil {
		return err
	}
	err = writen(writer, data)
	if err != nil {
		return err
	}
	return nil
}

