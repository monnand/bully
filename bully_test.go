package main

import (
	"testing"
	"net"
	"runtime"
)

func TestSingleBully(t *testing.T) {
	runtime.GOMAXPROCS(2)
	ln, err := net.Listen("tcp", ":8801")
	if err != nil {
		t.Errorf("%v\n", err)
	}
	bully := NewBully(ln, nil)
	err = bully.AddCandidate("127.0.0.1:8801", nil)
	if err != nil {
		t.Errorf("%v\n", err)
	}
	candy := bully.CandidateList()
	if len(candy) != 1 {
		t.Errorf("Wrong!")
	}
}

