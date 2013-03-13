package main

import (
	"net"
	"testing"
	//"runtime"
	"fmt"
	"time"
)

func TestSingleBully(t *testing.T) {
	//runtime.GOMAXPROCS(2)
	ln, err := net.Listen("tcp", ":8801")
	if err != nil {
		t.Errorf("%v\n", err)
	}
	bully := NewBully(ln, nil)
	err = bully.AddCandidate("127.0.0.1:8801", nil, 3*time.Second)
	if err != nil {
		t.Errorf("%v\n", err)
	}
	candy := bully.CandidateList()
	if len(candy) != 1 {
		t.Errorf("Wrong!")
	}
	for _, c := range candy {
		fmt.Printf("%v; %v\n", c.Addr, c.Id)
	}
	ln.Close()
}

func buildBullies(startPort, N int, t *testing.T) []*Bully {
	ret := make([]*Bully, 0, N)
	for i := 0; i < N; i++ {
		port := startPort + i
		addrStr := fmt.Sprintf("127.0.0.1:%v", port)
		ln, err := net.Listen("tcp", addrStr)
		if err != nil {
			t.Errorf("%v\n", err)
		}
		bully := NewBully(ln, nil)
		ret = append(ret, bully)
	}
	return ret
}

func buildConnections(bullies []*Bully, t *testing.T) {
	for _, alice := range bullies {
		for _, bob := range bullies {
			err := alice.AddCandidate(bob.myAddress().String(), nil, 3*time.Second)
			if err != nil {
				t.Errorf("%v\n", err)
			}
		}
	}
}

func testSameViewOnBullies(bullies []*Bully, t *testing.T) {
	for _, alice := range bullies {
		for _, bob := range bullies {

			bobCandy := bob.CandidateList()
			aliceCandy := alice.CandidateList()
			if len(bobCandy) != len(aliceCandy) {
				t.Errorf("Not same number of candidates!")
			}

			for _, bc := range bobCandy {
				found := false
				for _, ac := range aliceCandy {
					if bc.Id.Cmp(ac.Id) == 0 {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Bob's candidate %v cannot be found in Alice's", bc.Id)
				}
			}

		}
	}
}

func cleanBullies(bullies []*Bully) {
	for _, bully := range bullies {
		bully.Finalize()
	}
}

func TestDoubleBullyAuto(t *testing.T) {
	fmt.Printf("-------Double Bully Auto-------\n")
	bullies := buildBullies(8088, 2, t)
	fmt.Printf("-------Double Bully Auto Build Connections-------\n")
	buildConnections(bullies, t)
	testSameViewOnBullies(bullies, t)
	fmt.Printf("-------Double Bully Auto Clean-------\n")
	cleanBullies(bullies)
	fmt.Printf("-------Double Bully Auto Done-------\n")
}

func TestTripleBullyAuto(t *testing.T) {
	bullies := buildBullies(8088, 3, t)
	buildConnections(bullies, t)
	testSameViewOnBullies(bullies, t)
	cleanBullies(bullies)
}


func TestDoubleBully(t *testing.T) {
	fmt.Printf("-------Double Bully-------\n")
	aliceLn, err := net.Listen("tcp", ":8802")
	if err != nil {
		t.Errorf("%v\n", err)
	}
	alice := NewBully(aliceLn, nil)

	bobLn, err := net.Listen("tcp", ":8082")
	if err != nil {
		t.Errorf("%v\n", err)
	}
	if bobLn == nil {
		t.Errorf("WTF\n")
	}
	bobAddr := "127.0.0.1:8082"
	bob := NewBully(bobLn, nil)

	err = alice.AddCandidate(bobAddr, nil, 3*time.Second)
	if err != nil {
		t.Errorf("%v\n", err)
	}

	bobCandy := bob.CandidateList()
	aliceCandy := alice.CandidateList()
	if len(bobCandy) != len(aliceCandy) {
		t.Errorf("Should be 2 candidates!")
	}

	for _, bc := range bobCandy {
		found := false
		for _, ac := range aliceCandy {
			if bc.Id.Cmp(ac.Id) == 0 {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Bob's candidate %v cannot be found in Alice's", bc.Id)
		}
	}
	aliceLn.Close()
	bobLn.Close()
}
