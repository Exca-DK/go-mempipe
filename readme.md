Experimental one-way messaing through shared memory. Utilizes more resources but has way lower latency in compared to unix socket.
On my machine unix latency is around 90μs meanwhile in tests its within few.


memory structure:
```
                 uint32                           uint32
    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
    |        Sender Counter         |          Recv Counter         |
    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
    |      MsgSize + CodeSize       |              Code             |
    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
    |                                                               |
    |                            MESSAGE                            |
    |                                                               |
    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```



run test in processes as two seperate go instances

cd ./tests

go run test.go --id=1 --server

go run test.go --id=2

