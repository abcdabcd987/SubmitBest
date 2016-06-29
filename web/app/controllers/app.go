package controllers

import (
	"fmt"
	"log"
	"os"
	"path"
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
	/*var userSet map[string]bool
	var problemSet map[string]bool
	for _, a := range arg {
		if _, ok := userSet[a.username]; !ok {
			userSet[a.username] = true
		}
		if _, ok := problemSet[a.shortname]; !ok {
			problemSet[a.shortname] = true
		}
	}
	var usernames []string
	var shortnames []string
	for k := range userSet {
		usernames = append(usernames, k)
	}
	for k := range problemSet {
		shortnames = append(shortnames, k)
	}
	var cntUser int
	var cntProblem int
	model.DB.Model(&model.User{}).Where("username IN (?)", usernames).Count(&cntUser)
	if cntUser != len(usernames) {
		return fmt.Errorf("cntUser = %d != %d = len(usernames)", cntUser, len(usernames))
	}
	model.DB.Model(&model.Problem{}).Where("short_name IN (?)", shortnames).Count(&cntProblem)
	if cntProblem != len(shortnames) {
		return fmt.Errorf("cntProblem = %d != %d = len(shortnames)", cntProblem, len(shortnames))
	}

	// delete old
	empty := model.ProblemInput{}
	for _, a := range arg {
		model.DB.Where("username = ?, short_name = ?, testcase_id = ?",
			a.username, a.shortname, a.dataID).Delete(empty)
	}

	// create new*/
	for _, a := range arg {
		var old model.ProblemInput
		var err error
		ret := model.DB.Where("username = ? AND short_name = ? AND testcase_id = ?",
			a.username, a.shortname, a.dataID).First(&old)
		fmt.Printf("record not found: %v\n", ret.RecordNotFound())
		if !ret.RecordNotFound() && !a.reload {
			continue
		}

		filename := path.Join(a.shortname, sbest.RandString(16))
		os.MkdirAll(path.Join(SubmitBest.ROOT_USER_INPUT, a.shortname), 0755)
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
