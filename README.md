# election
## 개요
'election' is a Leader/Follower model middleware library. Basically, it is similar to Raft's method, but this library does not support functions such as log replication, and simply provides the upper layer whether a node is a leader or a follower. <br>
All communication is done with UDP protocol. Since broadcast communication is performed during the leader election process, it is burdensome to maintain TCP connections to all participant nodes. Also, since the log replication function is not used, the packet size is small (128 bytes), so there is no need to consider packet fragmentation. <br>

<br>
Every node manages an integer called Term, and nodes within the network must maintain the same Term. Only one leader must exist per term, and voting can only be done once. All messages within the network must be transmitted with their own terms. If the Term of the message sent by the other party is different from your Term, you must process it as follows.
