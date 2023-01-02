# election
'election' is a library that implements the Leader/Follower model middleware. Basically, it is similar to Raft, but this library does not support functions such as log replication, and simply provides the upper layer whether a node is a Leader or a Follower. Therefore, the role played by the Leader is freely determined externally. <br>
All communication is done with UDP protocol. Since broadcast communication is performed during the leader election process, it is burdensome to maintain TCP connections to all participant nodes. Also, since the log replication function is not used, the packet size is small (128 bytes), so there is no need to consider packet fragmentation. <br>

Every node manages an integer called Term, and nodes within the network must maintain the same Term. Only one leader must exist per term, and voting can only be done once. <br>

Node roles include Leader, Follower, Candidate and Shutdown. <br>

All nodes will participate in the Candidate role at the start. When a Candidate node is running, it waits for a random time (300-500 ms) and then broadcasts VoteMeMsg periodically. If a Candidate node receives a VoteMeMsg, it responds with a VoteMsg to the requester. You can vote only once per Term, and VoteMeMsg will not be broadcast if you have voted. If there is a node that has received a majority of VoteMsg, it broadcasts a NotifyLeaderMsg indicating that it has become a Leader node, and the node that received it converts to a Follower node. If no one becomes the Leader during the Election period, the Term is increased by one and VoteMeMsg is broadcast again. If a new node participates in a situation where a legitimate Leader exists in the network, it will transmit a VoteMeMsg, and the Leader node responds with its own Term when receiving the VoteMeMsg. When the Candidate node receives this response, it stops acting as a Candidate and participates as a Follower node. <br>

The Follower node periodically send ping to Leader node. If a ping timeout occurs, it broadcasts NewTermMsg and changes to a Candidate node. Other Follower nodes that receive the NewTermMsg immediately make a ping request to the Leader node to verify that it is not actually operating properly. If it works normally, it ignores it and performs the original task, and if it is confirmed that it does not work normally, it acts as a Candidate. When a Follower node receives a request that only a Leader node can perform, it relays the request to the Leader node. <br>

Leader nodes send pong replies to pings sent by Follower nodes, and count the number of ping messages received from Follower nodes to determine if they are isolated. If the number is less than a majority, it is changed to a Candidate node. The Follower nodes find out that the Leader node is not functioning properly, they change to Candidate nodes, and the new Leader election process proceeds. <br>

Regardless of the role, when all nodes receive NotifyLeaderMsg, they change their leader with the corresponding message and then perform Follower node operations.

## Message
All messages are less than 128 bytes in size. When sending a message to another node, the first 1 byte is the message type, and the remaining 127 bytes are filled with the actual payload. The sender/receiver must handle this properly.

```
0 1 2 3 4 5 6 7 8 9 . . . . . . . . . . . . . . . . . . . . . . 128
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| Type of message = 0 | Payload                                 |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```


### PingMsg
```go
PingMsg {}
```
Blah

### PongMsg
```go
PongMsg {
    Term   uint64
    Leader string
}
```
Blah