package controllers

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/abcdabcd987/SubmitBest"
	"github.com/abcdabcd987/SubmitBest/model"
	"github.com/abcdabcd987/SubmitBest/sbest"
	"github.com/abcdabcd987/SubmitBest/web/app/routes"
	"github.com/revel/revel"
	"golang.org/x/crypto/bcrypt"
)

type App struct {
	*revel.Controller
}

func (c App) Index() revel.Result {
	return c.Render()
}

func (c App) Register() revel.Result {
	return c.Render()
}

func (c App) Login() revel.Result {
	return c.Render()
}

func (c App) DoLogin(username, password string) revel.Result {
	var user model.User
	ret := model.DB.Where("username = ?", username).First(&user)
	if !ret.RecordNotFound() {
		hashed, _ := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if password == string(hashed) {
			c.Session["user.name"] = user.Username
			return c.Redirect(routes.App.Index())
		}
	}
	c.Flash.Error("Username or password incorrect.")
	c.FlashParams()
	return c.Redirect(routes.App.Login())
}

func (c App) DoRegister(user model.User, verifyPassword string) revel.Result {
	c.Validation.Required(verifyPassword)
	c.Validation.Required(user.Password == verifyPassword).Message("Password does not match")
	if c.Validation.HasErrors() {
		c.FlashParams()
		return c.Redirect(routes.App.Register())
	}

	hashed, _ := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	user.Password = string(hashed)
	if err := model.DB.Create(&user).Error; err != nil {
		c.Flash.Error("%v", err)
		c.FlashParams()
		return c.Redirect(routes.App.Register())
	}

	c.Session["user.name"] = user.Username
	return c.Redirect(routes.App.Index())
}

func (c App) Problem(shortname string) revel.Result {
	p := model.DB.Where("short_name = ?", shortname).First(&model.Problem{})
	if p.RecordNotFound() {
		return c.NotFound("Problem %s Not Found", shortname)
	}
	prob := p.Value
	return c.Render(prob)
}

func (c App) Management() revel.Result {
	return c.Render()
}

func (c App) RefreshProblem(shortname string) revel.Result {
	prob, err := sbest.ReadProblem(shortname)
	if err != nil {
		c.Flash.Error("%v", err)
		c.FlashParams()
		return c.Redirect(routes.App.Management())
	}

	var old model.Problem
	ret := model.DB.Where("short_name = ?", shortname).First(&old)
	if ret.RecordNotFound() {
		err = model.DB.Create(&prob).Error
	} else {
		err = model.DB.Model(&old).Updates(prob).Error
	}
	if err != nil {
		c.Flash.Error("%v", err)
		c.FlashParams()
		return c.Redirect(routes.App.Management())
	}

	c.Flash.Success("Problem %s refreshed", shortname)
	return c.Redirect(routes.App.Management())
}

func (c App) CreateContest(title, start, finish, problems string) revel.Result {
	st, _ := time.Parse(SubmitBest.TimeFormat, start)
	ed, _ := time.Parse(SubmitBest.TimeFormat, finish)
	if st.IsZero() || ed.IsZero() || st.After(ed) {
		c.Flash.Error("either start time or finish time is wrong.")
		c.FlashParams()
		return c.Redirect(routes.App.Management())
	}

	problemList := strings.Split(problems, "\n")
	for i, v := range problemList {
		problemList[i] = strings.TrimSpace(v)
	}

	var probs []model.Problem
	model.DB.Where("short_name IN (?)", problemList).Find(&probs)
	if len(probs) != len(problemList) {
		c.Flash.Error("given %d problems, find %d only.", len(problemList), len(probs))
		c.FlashParams()
		return c.Redirect(routes.App.Management())
	}

	contest := model.Contest{
		Title:    title,
		StartAt:  st,
		FinishAt: ed,
		Problems: probs,
	}
	if err := model.DB.Create(&contest).Error; err != nil {
		c.Flash.Error("%v", err)
		c.FlashParams()
		return c.Redirect(routes.App.Management())
	}

	c.Flash.Success("Contest added.")
	return c.Redirect(routes.App.Management())
}

type makeDataArg struct {
	shortname string
	username  string
	dataID    int
	reload    bool
}

func makeData(arg []makeDataArg) error {
	for _, a := range arg {
		var old model.ProblemInput
		var err error
		ret := model.DB.Where("username = ? AND short_name = ? AND testcase_id = ?",
			a.username, a.shortname, a.dataID).First(&old)
		fmt.Printf("record not found: %v\n", ret.RecordNotFound())
		if !ret.RecordNotFound() && !a.reload {
			continue
		}

		filename := path.Join(a.username, a.shortname, sbest.RandString(16))
		os.MkdirAll(path.Join(SubmitBest.ROOT_USER_INPUT, a.username, a.shortname), 0755)
		realpath := path.Join(SubmitBest.ROOT_USER_INPUT, filename)
		fmt.Printf("realpath=%s\n", realpath)
		fmt.Printf("a=%+v\n", a)
		sbest.MakeData(a.shortname, a.username, a.dataID, realpath)

		if ret.RecordNotFound() {
			now := model.ProblemInput{
				Username:   a.username,
				ShortName:  a.shortname,
				TestcaseID: a.dataID,
				InputFile:  filename,
			}
			err = model.DB.Create(&now).Error
			fmt.Printf("created\n")
		} else {
			old.InputFile = filename
			err = model.DB.Save(&old).Error
			fmt.Printf("updated\n")
		}
		if err != nil {
			log.Fatal(err)
		}
	}
	fmt.Printf("len(arg)=%d\n", len(arg))
	return nil
}

func (c App) MakeDataAll(shortname string, reload string) revel.Result {
	fmt.Printf("\n\n\n\n\n\nreload=%s\n", reload)
	var prob model.Problem
	if model.DB.Where("short_name = ?", shortname).First(&prob).RecordNotFound() {
		c.Flash.Error("Problem %s does not exist", shortname)
		c.FlashParams()
		return c.Redirect(routes.App.Management())
	}

	var users []model.User
	model.DB.Find(&users)

	args := make([]makeDataArg, 0, len(users)*prob.NumTestcase)
	for _, user := range users {
		for i := 1; i <= prob.NumTestcase; i++ {
			args = append(args, makeDataArg{
				shortname: shortname,
				username:  user.Username,
				dataID:    i,
				reload:    reload == "gen for all",
			})
		}
	}
	fmt.Printf("%+v\n", args)
	fmt.Printf("len(args)=%d\n", len(args))

	err := makeData(args)
	if err != nil {
		c.Flash.Error("%v", err)
		c.FlashParams()
		return c.Redirect(routes.App.Management())
	}

	c.Flash.Success("Successfully make data for problem %s for all users", shortname)
	return c.Redirect(routes.App.Management())
}

func (c App) Submit() revel.Result {
	return c.Render()
}

func (c App) DoSubmit() revel.Result {
	shortname := c.Params.Get("shortname")
	testcaseid := c.Params.Get("testcaseid")
	username, ok := c.Session["user.name"]
	if !ok {
		c.Flash.Error("You must login first!")
		c.FlashParams()
		return c.Redirect(routes.App.Login())
	}
	testcaseID, err := strconv.Atoi(testcaseid)
	if err != nil {
		c.Flash.Error("testcaseid invalid")
		c.FlashParams()
		return c.Redirect(routes.App.Submit())
	}
	var input model.ProblemInput
	ret := model.DB.Where("username = ? AND short_name = ? AND testcase_id = ?",
		username, shortname, testcaseID).First(&input)
	if ret.RecordNotFound() {
		c.Flash.Error("Testcase Not Found for User %s, Problem %s, TestcaseID %d", username, shortname, testcaseID)
		c.FlashParams()
		return c.Redirect(routes.App.Submit())
	}

	f1, ok1 := c.Params.Files["answer"]
	f2, ok2 := c.Params.Files["solution"]
	if !ok1 || !ok2 || len(f1) != 1 || len(f2) != 1 {
		c.Flash.Error("File upload incorrect.")
		c.FlashParams()
		return c.Redirect(routes.App.Submit())
	}
	answer := f1[0]
	solution := f2[0]
	pAnswer := path.Join(username, shortname, sbest.RandString(16)+path.Ext(answer.Filename))
	pSolution := path.Join(username, shortname, sbest.RandString(16)+path.Ext(solution.Filename))
	iAnswer, err1 := answer.Open()
	iSolution, err2 := solution.Open()
	os.MkdirAll(path.Dir(path.Join(SubmitBest.ROOT_USER_ANSWER, pAnswer)), 0755)
	os.MkdirAll(path.Dir(path.Join(SubmitBest.ROOT_USER_SOLUTION, pSolution)), 0755)
	oAnswer, err3 := os.Create(path.Join(SubmitBest.ROOT_USER_ANSWER, pAnswer))
	oSolution, err4 := os.Create(path.Join(SubmitBest.ROOT_USER_SOLUTION, pSolution))
	defer iAnswer.Close()
	defer iSolution.Close()
	defer oAnswer.Close()
	defer oSolution.Close()
	_, err5 := io.Copy(oAnswer, iAnswer)
	_, err6 := io.Copy(oSolution, iSolution)
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil || err6 != nil {
		if err1 != nil {
			c.Flash.Error("%v", err1)
			fmt.Println("1")
		}
		if err2 != nil {
			c.Flash.Error("%v", err2)
			fmt.Println("2")
		}
		if err3 != nil {
			c.Flash.Error("%v", err3)
			fmt.Println("3")
		}
		if err4 != nil {
			c.Flash.Error("%v", err4)
			fmt.Println("4")
		}
		if err5 != nil {
			c.Flash.Error("%v", err5)
			fmt.Println("5")
		}
		if err6 != nil {
			c.Flash.Error("%v", err6)
			fmt.Println("6")
		}
		c.FlashParams()
		return c.Redirect(routes.App.Submit())
	}

	submit := model.Submit{
		Username:     username,
		ShortName:    shortname,
		TestcaseID:   testcaseID,
		InputFile:    input.InputFile,
		SolutionFile: pSolution,
		AnswerFile:   pAnswer,
	}
	if err := model.DB.Create(&submit).Error; err != nil {
		c.Flash.Error("%v", err)
		c.FlashParams()
		return c.Redirect(routes.App.Submit())
	}

	return c.Redirect(routes.App.Submit())
}
