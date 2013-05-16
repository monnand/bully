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
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

var argvPort = flag.Int("port", 8117, "port to listen")
var argvCandidates = flag.String("nodes", "", "comma separated list of nodes.")
var argvRestBind = flag.String("http", "127.0.0.1:8080", "Network address which will be bind to a restful service")
var argvShowPort = flag.Bool("showport", false, "Output the leader's port number (which is only useful for debug purpose)")
var argvUnixTime = flag.Bool("unixTime", true, "Show the timestamp in unix time")
var argvWebHookURL = flag.String("url", "none", "URL of the webhook. The leader will post an HTTP request to this URL when it has been elected")

func main() {
	flag.Parse()
	bindAddr := fmt.Sprintf("0.0.0.0:%v", *argvPort)

	ln, err := net.Listen("tcp", bindAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}
	hook := new(WebObserver)
	hook.URL = *argvWebHookURL
	hook.Default = 200
	hook.Timeout = 3 * time.Second

	bully := NewBully(ln, nil, hook)
	defer bully.Finalize()

	nodeAddr := strings.Split(*argvCandidates, ",")
	dialTimtout := 5 * time.Second

	for _, node := range nodeAddr {
		if len(node) == 0 {
			continue
		}
		err := bully.AddCandidate(node, nil, dialTimtout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v cannot be added: %v\n", node, err)
		}
	}

	fmt.Printf("My ID: %v\n", bully.MyId())

	web := NewWebAPI(bully, *argvShowPort, *argvUnixTime)
	web.Run(*argvRestBind)
	bully.Finalize()
}
