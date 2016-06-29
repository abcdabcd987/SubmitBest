package sbest

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/abcdabcd987/SubmitBest"
	"github.com/abcdabcd987/SubmitBest/model"
	"github.com/russross/blackfriday"
)

func ReadProblem(shortname string) (*model.Problem, error) {
	// read config
	pConf := path.Join(SubmitBest.ROOT_PROBLEM, shortname, "config")
	file, err := os.Open(pConf)
	defer file.Close()
	if err != nil {
		return nil, err
	}

	const timeFormat = "2006-01-02 15:04:03"
	var title string
	var numTestcase int
	var secretBefore time.Time

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		splited := strings.SplitN(scanner.Text(), ":", 2)
		if len(splited) != 2 {
			return nil, fmt.Errorf("bad key/value pair `%s`", splited)
		}
		key := strings.TrimSpace(splited[0])
		value := strings.TrimSpace(splited[1])
		switch key {
		case "title":
			title = value
		case "numTestcase":
			numTestcase, _ = strconv.Atoi(value)
		case "secretBefore":
			secretBefore, _ = time.Parse(timeFormat, value)
		}
	}

	var errMessage string
	if title == "" {
		errMessage = "title is empty"
	}
	if numTestcase == 0 {
		errMessage = "numTestcase is 0"
	}
	if secretBefore.IsZero() {
		errMessage = "secretBefore is empty"
	}
	if errMessage != "" {
		return nil, fmt.Errorf("%v", errMessage)
	}

	// read problem
	pProblem := path.Join(SubmitBest.ROOT_PROBLEM, shortname, "problem.md")
	md, err := ioutil.ReadFile(pProblem)
	if err != nil {
		return nil, err
	}
	html := blackfriday.MarkdownCommon(md)

	return &model.Problem{
		ShortName:    shortname,
		Title:        title,
		Description:  string(html),
		SecretBefore: secretBefore,
		NumTestcase:  numTestcase,
	}, nil
}

var lock sync.Mutex
var problemPath = make(map[string]string)

func Prepare(shortname string, reload bool, lockNow bool) {
	if lockNow {
		lock.Lock()
		defer lock.Unlock()
	}
	if _, ok := problemPath[shortname]; ok && !reload {
		return
	}

	pTemp, _ := ioutil.TempDir("", "prepare")
	pProblem := path.Join(SubmitBest.ROOT_PROBLEM, shortname)
	exec.Command("cp", "-r", pProblem, pTemp).Run()
	pTemp = path.Join(pTemp, shortname)
	problemPath[shortname] = pTemp
	exec.Command("bash", path.Join(pTemp, "prepare.bash")).Run()
}

func MakeData(shortname, username string, dataID int, outputPath string) {
	Prepare(shortname, false, true)
	lock.Lock()
	defer lock.Unlock()

	pProblem := problemPath[shortname]
	fmt.Printf("pProblem=%s\n", pProblem)
	fmt.Printf("username=%s, dataID=%d\n", username, dataID)
	exec.Command("bash", path.Join(pProblem, "makedata.bash"),
		username, strconv.Itoa(dataID)).Run()
	exec.Command("mv", path.Join(pProblem, "data.in"), outputPath).Run()
}

func Judge(shortname, username string, dataID int, inputPath, answerPath string) (int, string) {
	Prepare(shortname, false, true)
	lock.Lock()
	defer lock.Unlock()

	pProblem := problemPath[shortname]
	exec.Command("bash", path.Join(pProblem, "judge.bash"),
		username, strconv.Itoa(dataID), inputPath, answerPath).Run()
	scoreContent, _ := ioutil.ReadFile(path.Join(pProblem, "score.txt"))
	score, _ := strconv.Atoi(strings.TrimSpace(string(scoreContent)))
	msg, _ := ioutil.ReadFile(path.Join(pProblem, "message.txt"))
	os.Remove(path.Join(pProblem, "score.txt"))
	os.Remove(path.Join(pProblem, "message.txt"))
	return score, string(msg)
}
