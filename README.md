# election
## 개요
'election'은 Leader/Follwer 모델 미들웨어 라이브러리입니다. 기본적으로 Raft의 방식과 비슷하지만 이 라이브러리는 로그 복제와 같은 기능은 지원하지 않으며, 단순히 윗 레이어에게 노드가 Leader인지 Follower인지에 대한 여부만 제공합니다. <br>
모든 통신은 UDP 프로토콜로 이루어집니다. 리더 선출 과정에서 broadcast 통신을 하기 때문에 모든 참가자 노드에 대한 TCP connection은 유지하는 것은 부담스럽습니다. 또한 로그 복제 기능을 사용하지 않기 때문에 패킷 크기가 작아 패킷 단편화와 같은 고려를 할 필요도 없습니다. <br>

<br>
모든 노드는 Term이라는 정수를 관리하며, 네트워크 내 노드는 동일한 Term을 유지해야 합니다. 한 Term에 하나의 Leader만이 존재해야 하고, 투표 또한 한번만 할 수 있습니다. 네트워크 내 모든 메세지는 자신의 Term 을 넣어 전송해야 합니다. 만약 상대가 보낸 메세지의 Term과, 나의 Term이 다르다면 아래와 같이 처리해야 합니다. <br>

- (local.Term == msg.Term): 정상
- (local.Term < msg.Term): 비정상, 나의 Term이 뒤쳐져 있습니다. 다른 노드들은 별개의 Term을 진행하고 있습니다. 다른 노드로부터 현재 Term과 Leader 노드 정보를 가져와 commit 합니다.
- (local.Term > msg.Term): 비정상, 상대방 노드의 Term이 뒤쳐져 있습니다. 메세지를 무시합니다. 상대방 노드는 메세지에 대한 응답을 받지 못하고 무언가 잘못됐다고 판단합니다. 노드는 리더 재선출 과정을 진행하고, 자신만 뒤쳐져 있다는 것을 알게 되어 복구 과정에 들어갑니다.


## 메세지 타입

### PingMsg
```go
PingMsg {
}
```
사용자가 처음에 클러스터에 가입할 때나, Follower가 Leader의 작동 여부를 확인할 때 사용하는 메세지 타입입니다. 역할을 떠나 모든 노드는 PingMsg에 대해 PongMsg로 응답해야 합니다. Ping에 대한 타임아웃 또한 명세되어야 하며, 해당 시간이 지나면 상대 노드가 작동하지 않는다고 판단합니다.
### PongMsg
```go
PongMsg {
    Term   uint64
    Leader string
}
```
PingMsg에 대한 응답입니다. 노드의 최근 Term과 현재 Leader의 주소를 포함합니다. PongMsg를 받은 노드는 자신의 Term과 비교합니다. 만약 자신의 Term과 동일하다면 정상이며, 자신의 Term보다 작다면 노드가 뒤쳐져 있는 상태입니다. 때문에 메세지 안 Term과 Leader를 저장해야 합니다. <br>
이 메세지는 주로 아래와 같은 경우에 수신하게 됩니다. <br>

1. Follwer가 Leader로 부터 받은 응답
2. NewTermMsg에 대한 응답

만약 NewTermMsg를 받으면, Follower는 즉시 Leader에게 PingMsg를 보냅니다(상대 노드만 Leader에 접근하지 못할 가능성 존재). 만약 Leader로부터 PongMsg를 정상적으로 수신한다면 해당 요청을 무시하며, PongMsg를 받지 못한다면 NewTerm 작업을 진행합니다.

만약 자신의 Term이 메세지 안 Term 보다 작다면 단순히 무시하도록 합니다.

### NewTermMsg
```go
NewTermMsg {
    Term   uint64
}
```
Ping-Pong 과정을 통하여 Leader 노드가 정상 작동하지 않음을 감지한 Follower는 네트워크에 NewTermMsg를 브로드캐스팅 합니다. 이 메세지를 받은 노드는 Leader에게 Ping을 요청하여 재확인을 해야하며, 수신자 또한 Leader가 정상작동하지 않는다는 판단을 하게 된다면 아래와 같은 작업을 수행해야 합니다.

1. Follwer 노드 작업을 중지합니다.
2. Term을 1 증가시킨 뒤 Term에 대한 투표권을 1회 부여합니다.
3. 일정 시간을 대기한 뒤(100~300 ms), VoteMeMsg를 브로드캐스팅 합니다.
4. 만약 VoteMeMsg를 받는다면, VoteMsg로 응답한 뒤 투표권을 해제 시킵니다(한 Term에 1회만 투표해야 합니다).
5. 대다수 노드의 VoteMsg를 받는다면 LeaderNotifyMsg를 브로드캐스팅 합니다. 그 후 Leader 노드 작업을 실행합니다.
6. 만약 LeaderNotifyMsg를 받는다면 위 과정을 중지한 뒤, 전달 받은 Leader, Term 데이터로 업데이트하며 Follower 노드 작업을 실행합니다.
7. 일정 기간 동안 LeaderNotifyMsg를 받지 못한다면, 2번 작업부터 재시도 합니다. 

만약 NewTermMsg를 보낸 후, 일정 기간동안 다른 노드로부터 VoteMeMsg, VoteMsg, LeaderNotifyMsg 메세지를 받지 못한다면 네트워크 초기 참여했을 때와 동일한 프로세스를 실행합니다(Ping 브로드캐스팅).
### AddMemberMsg
```go
AddMemberMsg {
    Term   uint64
    Member string
}
```
메세지 안 Member를 네트워크에 추가합니다. 해당 작업은 Leader 노드만 수행합니다. 만약 Follower 노드가 AddMemberMsg를 받는다면 Leader 노드로 Relay 해야 합니다.
Memberlist는 동기화 되지 않으며, Leadaer가 변경되었을 때 Follower는 다시 AddMemberMsg를 요청해야 합니다. 일반적으로 LeaderNotifyMsg를 받았을 때 Leader를 새롭게 설정하고 AddMemberMsg를 전송해줘 업데이트가 되도록 합니다.

### DelMemberMsg (예정)
```go
DelMemberMsg {
    Term   uint64
    Member string
}
```
메세지 안 Member를 네트워크에서 제거합니다. 해당 작업은 Leader 노드만 수행합니다. 만약 Follower 노드가 DelMemberMsg를 받는다면 Leader 노드로 Relay 해야 합니다.
Memberlist는 동기화 되지 않습니다.

### VoteMeMsg
```go
VoteMeMsg {
    Term   uint64
}
```
### VoteMsg
```go
VoteMsg {
    Term   uint64
}
```
### LeaderNotifyMsg
```go
LeaderNotifyMsg {
    Term   uint64
    Leader string
}
```


