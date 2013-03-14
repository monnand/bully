bully
=====

Bully algorithm implemented in Go.

**NOTE:** This program is indented to be used within LAN among small number of
nodes. It is vulnerable to be exposed to outside network and it may consumes a
lot of bandwith on a large cluster.

## Get started

- Download and install:

        go get github.com/monnand/bully

- Suppose we want to run the program on two machines whose IPs are 192.168.1.67
  and 192.168.1.68, respectively.
- Run the following command on both machines:

        bully -port=8117 -nodes="192.168.1.67:8117,192.168.1.68:8117" -rest=0.0.0.0:8080

- To know who is the leader, use HTTP to connect one of the machines with path */leader*:

        curl http://192.168.1.67:8080/leader

## Details

Let's start with a real example. 


