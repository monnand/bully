package main

import (
	"fmt"
	"net/http"
	"os"
)

type WebAPI struct {
	bully *Bully
}

const (
	newCandidate = "/join"
	getLeader    = "/leader"
)

func NewWebAPI(bully *Bully) *WebAPI {
	ret := new(WebAPI)
	ret.bully = bully
	return ret
}

func (self *WebAPI) join(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Success\r\n")
}

func (self *WebAPI) leader(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Success\r\n")
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
