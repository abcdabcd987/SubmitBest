package sbest

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/abcdabcd987/SubmitBest"
	"github.com/abcdabcd987/SubmitBest/model"
)

func getTask() *model.Submit {
	var r GetTaskReply
	ok := Call(SubmitBest.TASKSERVER_ADDR, "TaskServer.GetTask", GetTaskArg{}, &r)
	if ok && r.Found {
		return &r.Submit
	}
	return nil
}

func downloadFile(remote, local string) {
	fmt.Printf("downloadFile(%s,%s)\n", remote, local)
	cmd := exec.Command("curl", "-o", local, remote)
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}

func judgeLoop() {
	fmt.Printf("judgeloop\n")
	task := getTask()
	if task == nil {
		return
	}

	fmt.Printf("got a task: %+v\n", task)

	input, _ := ioutil.TempFile("", "input")
	answer, _ := ioutil.TempFile("", "answer")
	defer os.Remove(input.Name())
	defer os.Remove(answer.Name())

	downloadFile(SubmitBest.FILESERVER_ADDR+"input/"+task.InputFile, input.Name())
	downloadFile(SubmitBest.FILESERVER_ADDR+"answer/"+task.AnswerFile, answer.Name())
	score, msg := Judge(task.ShortName, task.Username, task.TestcaseID, input.Name(), answer.Name())

	fmt.Printf("    judge finish\n")

	task.Score = score
	task.Message = msg
	task.Status = "finished"
	for {
		ok := Call(SubmitBest.TASKSERVER_ADDR, "TaskServer.JudgeResult",
			JudgeResultArg{*task}, &JudgeResultReply{})
		if ok {
			break
		}
	}

	fmt.Printf("    update finish")
}

func reloadAll(lockNow bool) {
	if lockNow {
		lock.Lock()
		defer lock.Unlock()
	}
	files, err := ioutil.ReadDir(SubmitBest.ROOT_PROBLEM)
	if err != nil {
		log.Fatal(err)
		return
	}

	fmt.Printf("reload all problems\n")

	for _, file := range files {
		name := file.Name()
		if name == ".git" {
			continue
		}
		fmt.Printf("    reloading %s...\n", name)
		Prepare(name, true, false)
	}

	fmt.Printf("    all done\n")
}

func refreshLoop() {
	lock.Lock()
	defer lock.Unlock()

	var out bytes.Buffer
	cmd := exec.Command("bash", "-c", fmt.Sprintf("cd %s; git pull;", SubmitBest.ROOT_PROBLEM))
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		fmt.Println(err)
		return
	}
	if strings.TrimSpace(out.String()) == "Already up-to-date." {
		return
	}

	reloadAll(false)
}

func JudgeMain() {
	reloadAll(true)

	go func() {
		for {
			judgeLoop()
			time.Sleep(100 * time.Millisecond)
		}
	}()

	go func() {
		for {
			refreshLoop()
			time.Sleep(1 * time.Second)
		}
	}()

	ch := make(chan int)
	<-ch
}
