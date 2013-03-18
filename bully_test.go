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
	"net"
	"testing"
	//"runtime"
	"fmt"
	"time"
)

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

func printCandies(bullies []*Bully) {
	for _, alice := range bullies {
		fmt.Printf("[LISTCANDYLIST] %v:\n", alice.localhost())
		aliceCandy := alice.CandidateList()
		for _, c := range aliceCandy {
			fmt.Printf("\t%v\n", c.Addr)
		}
	}
}

func buildConnections(bullies []*Bully, t *testing.T) {
	for _, alice := range bullies {
		fmt.Println("++++++++++++++++")
		for _, bob := range bullies {
			fmt.Printf("[TEST] Ask %v to add %v\n", alice.localhost(), bob.localhost())
			err := alice.AddCandidate(bob.localhost().String(), nil, 3*time.Second)
			if err != nil {
				t.Errorf("%v\n", err)
			}
			fmt.Printf("[TEST] %v has added%v\n", alice.localhost(), bob.localhost())
			printCandies(bullies)
			fmt.Println("#############")
		}
		fmt.Println("++++++++++++++++")
	}
}

func testSameViewOnBullies(bullies []*Bully, t *testing.T) {
	n := len(bullies)
	for _, alice := range bullies {
		for _, bob := range bullies {

			bobCandy := bob.CandidateList()
			aliceCandy := alice.CandidateList()
			if len(bobCandy) != len(aliceCandy) {
				t.Errorf("Not same number of candidates!")
			}

			if len(bobCandy) != n {
				t.Errorf("Should have %v candidates; Now have %v", n, len(bobCandy))
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

func testSameLeader(bullies []*Bully, t *testing.T) {
	for _, alice := range bullies {
		for _, bob := range bullies {
			aliceLeader, err := alice.Leader()
			if err != nil {
				t.Errorf("%v\n", err)
			}
			bobLeader, err := bob.Leader()
			if err != nil {
				t.Errorf("%v\n", err)
			}

			if aliceLeader.Id.Cmp(bobLeader.Id) != 0 {
				t.Errorf("%v thinks its leader is %v;\n\tbut %v thinks its leader is %v\n",
					alice.MyId(), aliceLeader.Id, bob.MyId(), bobLeader.Id)
			}
		}
	}
}

func cleanBullies(bullies []*Bully) {
	for _, bully := range bullies {
		bully.Finalize()
	}
}

func TestSingleBullyElect(t *testing.T) {
	fmt.Println("-------------------1------------------")
	bullies := buildBullies(8088, 1, t)
	for _, b := range bullies {
		fmt.Printf("bully: %v on addr %v\n", b.MyId(), b.MyAddr())
	}
	buildConnections(bullies, t)
	testSameViewOnBullies(bullies, t)
	testSameLeader(bullies, t)
	cleanBullies(bullies)
	fmt.Println("-------------------#------------------")
}

func TestDoubleBullyElect(t *testing.T) {
	fmt.Println("-------------------2------------------")
	bullies := buildBullies(8088, 2, t)
	for _, b := range bullies {
		fmt.Printf("bully: %v on addr %v\n", b.MyId(), b.MyAddr())
	}
	buildConnections(bullies, t)
	testSameViewOnBullies(bullies, t)
	testSameLeader(bullies, t)
	cleanBullies(bullies)
	fmt.Println("-------------------#------------------")
}

func TestTripleBullyElect(t *testing.T) {
	fmt.Println("-------------------3------------------")
	bullies := buildBullies(8088, 3, t)
	for _, b := range bullies {
		fmt.Printf("bully: %v on addr %v\n", b.MyId(), b.MyAddr())
	}
	buildConnections(bullies, t)
	testSameViewOnBullies(bullies, t)
	testSameLeader(bullies, t)
	cleanBullies(bullies)
	fmt.Println("-------------------#------------------")
}

func Test4BullyElect(t *testing.T) {
	fmt.Println("-------------------4------------------")
	bullies := buildBullies(8088, 4, t)
	for _, b := range bullies {
		fmt.Printf("bully: %v on addr %v\n", b.MyId(), b.MyAddr())
	}
	buildConnections(bullies, t)
	testSameViewOnBullies(bullies, t)
	testSameLeader(bullies, t)
	cleanBullies(bullies)
	fmt.Println("-------------------#------------------")
}

