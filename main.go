package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

var argvMyAddress = flag.String("addr", "127.0.0.1:8117", "Public address of this host in the format of IP:Port. Example: 192.168.1.10:8117")

func main() {
	flag.Parse()
	myAddr, err := net.ResolveTCPAddr("tcp", *argvMyAddress)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid address %v: %v\n", *argvMyAddress, err)
		return
	}
	sa := strings.Split(*argvMyAddress, ":")
	if len(sa) < 2 {
		fmt.Fprintf(os.Stderr, "The public address must contain the port number. IP:Port\n")
		return
	}
	_, err = strconv.Atoi(sa[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "The port number has to be an integer: %v\n", err)
		return
	}
	bindAddr := fmt.Sprintf("0.0.0.0:%v", sa[1])

	fmt.Printf("Bind addr: %v\n", bindAddr)
	fmt.Printf("Public address: %v\n", myAddr)

	ln, err := net.Listen("tcp", bindAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}
	bully := NewBully(ln, nil)
	web := NewWebAPI(bully)
	web.Run("127.0.0.1:8080")
}
