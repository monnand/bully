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
	"fmt"
	"net/http"
	"os"
	"strings"
)

type WebAPI struct {
	bully *Bully
	showPort bool
}

const (
	newCandidate = "/join"
	getLeader    = "/leader"
)

func NewWebAPI(bully *Bully, showPort bool) *WebAPI {
	ret := new(WebAPI)
	ret.bully = bully
	ret.showPort = showPort
	return ret
}

func (self *WebAPI) join(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Not implemented\r\n")
}

func (self *WebAPI) leader(w http.ResponseWriter, r *http.Request) {
	leader, err := self.bully.Leader()
	if err != nil {
		fmt.Fprint(w, "Error: %v\r\n", err)
	}
	var leaderAddr string
	if self.bully.MyId().Cmp(leader.Id) == 0 {
		if len(leader.Addr) == 0 {
			fmt.Fprintf(w, "me\r\n")
			return
		} else {
			leaderAddr = leader.Addr
		}
	} else {
		leaderAddr = leader.Addr
	}

	if !self.showPort {
		ae := strings.Split(leaderAddr, ":")
		if len(ae) > 1 {
			leaderAddr = strings.Join(ae[:len(ae) - 1], ":")
		}
	}
	fmt.Fprintf(w, "%v\r\n", leaderAddr)
}

func (self *WebAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	switch r.URL.Path {
	case newCandidate:
		self.join(w, r)
	case getLeader:
		self.leader(w, r)
	}
}

func (self *WebAPI) Run(addr string) {
	http.Handle(newCandidate, self)
	http.Handle(getLeader, self)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
}

