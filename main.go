package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"time"
	"strings"
)

var argvPort = flag.Int("port", 8117, "port to listen")
var argvCandidates = flag.String("nodes", "", "comma separated list of nodes.")
var argvRestBind = flag.String("rest", "127.0.0.1:8080", "Network address which will be bind to a restful service")

func main() {
	flag.Parse()
	bindAddr := fmt.Sprintf("0.0.0.0:%v", *argvPort)
	fmt.Printf("Bind addr: %v\n", bindAddr)

	ln, err := net.Listen("tcp", bindAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}
	bully := NewBully(ln, nil)

	nodeAddr := strings.Split(*argvCandidates, ",")
	dialTimtout := 5 * time.Second

	for _, node := range nodeAddr {
		err := bully.AddCandidate(node, nil, dialTimtout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v cannot be added: %v\n", node, err)
		}
	}

	web := NewWebAPI(bully)
	web.Run(*argvRestBind)
}

