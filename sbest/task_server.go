package sbest

import (
	"log"
	"net"
	"net/rpc"
	"sync"

	"github.com/abcdabcd987/SubmitBest/model"
)

type TaskServer struct {
	sync.Mutex
}

func RunTaskServer(addr string) {
	ts := new(TaskServer)

	rpcs := rpc.NewServer()
	rpcs.Register(ts)

	l, e := net.Listen("tcp", addr)
	if e != nil {
		log.Fatal("listen error", e)
	}

	// RPC handler
	go func() {
		for {
			conn, err := l.Accept()
			if err == nil {
				go func() {
					rpcs.ServeConn(conn)
					conn.Close()
				}()
			} else {
				log.Fatal("RegistrationServer: accept error", err)
				break
			}
		}
		log.Fatal("RegistrationServer: done\n")
	}()
}

type HelloArg struct {
	Name string
}

type HelloReply struct {
	Message string
}

func (ts *TaskServer) Hello(args HelloArg, reply *HelloReply) error {
	reply.Message = "Hello " + args.Name
	return nil
}

type GetTaskArg struct {
}

type GetTaskReply struct {
	Found  bool
	Submit model.Submit
}

func (ts *TaskServer) GetTask(args GetTaskArg, reply *GetTaskReply) error {
	ts.Lock()
	defer ts.Unlock()
	ret := model.DB.Where("status = ?", "pending").Order("id ASC").First(&reply.Submit)
	reply.Found = !ret.RecordNotFound()
	if ret.RecordNotFound() {
		return nil
	}

	reply.Submit.Status = "judging"
	if err := model.DB.Save(reply.Submit).Error; err != nil {
		log.Fatal(err)
	}
	return nil
}

type JudgeResultArg struct {
	Submit model.Submit
}

type JudgeResultReply struct {
}

func (ts *TaskServer) JudgeResult(args JudgeResultArg, reply *JudgeResultReply) error {
	ts.Lock()
	defer ts.Unlock()
	if err := model.DB.Save(args.Submit).Error; err != nil {
		log.Fatal(err)
	}
	return nil
}
