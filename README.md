bully
=====

A leader election program written in [Go](http://golang.org)(golang) using
[Bully leader election
algorithm](http://en.wikipedia.org/wiki/Bully_algorithm).

**NOTE:** This program is indented to be used within LAN among small number of
nodes. It is vulnerable to attack if it is exposed to outside network and it
may consume a lot of bandwith on a large cluster.

## Quick start

- Download and install:

        go get github.com/monnand/bully

- Suppose we want to run the program on two machines whose IPs are 192.168.1.67
  and 192.168.1.68, respectively.
- Run the following command on both machines:

        bully -port=8117 -nodes="192.168.1.67:8117,192.168.1.68:8117" -http=0.0.0.0:8080

- To know who is the leader, use HTTP to connect one of the machines with path */leader*:

        curl http://192.168.1.67:8080/leader

## A real example

### Starting from a single node

Let's start with a real example. Again, we have two nodes at first, 192.168.1.67 and 192.168.1.68

On 192.168.1.67, I first execute the following command:

        bully -port=8117 -nodes="192.168.1.67:8117,192.168.1.68:8117" -http=0.0.0.0:8080

This command contains three arguments, *port*, *nodes* and *http*.
- *port*: tells *bully* to listen on port 8117 waiting for other candidates'
  connection.
- *http*: tells *bully* to set up a web service on port 8080 accepting
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

        bully -port=8117 -nodes="192.168.1.67:8117,192.168.1.68:8117" -http=0.0.0.0:8080

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

        bully -port=8117 -nodes="192.168.1.67:8117,192.168.1.68:8117,192.168.1.69:8117" -http=0.0.0.0:8080

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

### Killing the leader

What if the leader node crashed? The rest of the candidates will elect a new
leader on the fly.

Suppose we killed the leader in previours example and ask who is the leader to
the rest of the group:

        $ curl http://192.168.1.67:8080/leader
        192.168.1.67
        $ curl http://192.168.1.69:8080/leader
        192.168.1.67

If the old leader recovered, then it can join the group again by executing the
following command:

        bully -port=8117 -nodes="192.168.1.67:8117,192.168.1.68:8117,192.168.1.69:8117" -http=0.0.0.0:8080

## Typical scenario

The good part of *bully* is that once we decided the set of candidates, we can
use exactly the same command with same parameters on any candidate node to
start *bully*. This is very nice because we can use the same virtual machine
image on the cloud.

Imaging we have a service need a node to do some management. Such work may not
be heavy but critical. So we need to have some backup servers in case of the
manager node down. Normally, 3 or 5 node will be enough. Since all nodes are
same, we need to select a leader node to do the job, let others running and
wait the leader die. Then we can use *bully* here to decide who is the leader,
and elect another leader when the old one down or new one comes.

