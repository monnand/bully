bully
=====

Bully algorithm implemented in Go.

**NOTE:** This program is indented to be used within LAN among small number of
nodes. It is vulnerable to be exposed to outside network and it may consumes a
lot of bandwith on a large cluster.

## Quick start

- Download and install:

        go get github.com/monnand/bully

- Suppose we want to run the program on two machines whose IPs are 192.168.1.67
  and 192.168.1.68, respectively.
- Run the following command on both machines:

        bully -port=8117 -nodes="192.168.1.67:8117,192.168.1.68:8117" -rest=0.0.0.0:8080

- To know who is the leader, use HTTP to connect one of the machines with path */leader*:

        curl http://192.168.1.67:8080/leader

## Real example

### Starting from a single node

Let's start with a real example. Again, we have two nodes at first, 192.168.1.67 and 192.168.1.68

On 192.168.1.67, I first execute the following command:

        bully -port=8117 -nodes="192.168.1.67:8117,192.168.1.68:8117" -rest=0.0.0.0:8080

This command contains three arguments, *port*, *nodes* and *rest*.
- *port*: tells *bully* to listen on port 8117 waiting for other candidates'
  connection.
- *rest*: tells *bully* to set up a web service on port 8080 accepting
  connections from any client. The client will ask who is the leader using
  HTTP.
- *nodes*: tells *bully* the address (IP and port) of the initial set of candidates.

Note that by executing the command above, *bully* will complain by printing the
following message at begining: 

        192.168.1.68:8117 cannot be added: dial tcp 192.168.1.68:8117: connection refused

This message says that it cannot connect to 192.168.1.68, which is correct
because we haven't started it yet.

Now, let's ask it who is the leader:

        $ curl http://192.168.1.67:8080/leader
        192.168.1.67

The returned value contains the IP address of the elected leader. In this case,
there is only one candidate, and apparently, it is the leader.

### Adding one more node

Now let's move on to 192.168.1.68 and start another *bully* instance with same command:

        bully -port=8117 -nodes="192.168.1.67:8117,192.168.1.68:8117" -rest=0.0.0.0:8080

Since both candidates specified in *nodes* argument are on-line, there is no more complain.

Then we can ask both 192.168.1.67 and 192.168.1.68 who is the leader. And we
can always get the same answer:

        $ curl http://192.168.1.67:8080/leader
        192.168.1.68
        $ curl http://192.168.1.68:8080/leader
        192.168.1.68

The leader now is 192.168.1.68. In this case, they elected another node as the
leader. But this is not always the case. Every time a new node join, there will
be a new election and the winner of the election may or may not be the
previours leader.

### Playing with three nodes

Let's say we want to add one more node into this cluster, whose IP is
192.168.1.69. Only thing we need to change is to start a new instance of
*bully* on 192.168.1.69 with almost the same command:

        bully -port=8117 -nodes="192.168.1.67:8117,192.168.1.68:8117,192.168.1.69:8117" -rest=0.0.0.0:8080

The only changed part is the *nodes* parameter. Since we added one more node,
then we need to add it as an initial candidate. **The good part is that you don't
need to stop the other two instances running on 192.168.1.67 and 192.168.1.68.**
They will automatically update the candidate list and elect another leader.

Once the new instance started, we can query any of one of the *bully* instance
about the leader using HTTP:

        $ curl http://192.168.1.67:8080/leader
        192.168.1.68
        $ curl http://192.168.1.68:8080/leader
        192.168.1.68
        $ curl http://192.168.1.69:8080/leader
        192.168.1.68

This time, the leader is same as previours leader, which is 192.168.1.68.


