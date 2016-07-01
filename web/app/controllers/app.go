package controllers

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"sort"
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

func (c App) AddUser() revel.Result {
	if user := c.connected(); user != nil {
		c.RenderArgs["user"] = user
	}
	return nil
}

func (c App) connected() *model.User {
	if c.RenderArgs["user"] != nil {
		return c.RenderArgs["user"].(*model.User)
	}
	if username, ok := c.Session["user.name"]; ok {
		var user model.User
		ret := model.DB.Where("username=?", username).First(&user)
		if !ret.RecordNotFound() {
			return &user
		}
	}
	return nil
}

func (c App) Index() revel.Result {
	return c.Render()
}

func (c App) Register() revel.Result {
	_, ok := c.Session["user.name"]
	if ok {
		c.Flash.Error("You already logged in.")
		return c.Redirect(routes.App.Index())
	}
	return c.Render()
}

func (c App) Login() revel.Result {
	_, ok := c.Session["user.name"]
	if ok {
		c.Flash.Error("You already logged in.")
		return c.Redirect(routes.App.Index())
	}
	return c.Render()
}

func (c App) ModifyUser() revel.Result {
	username, ok := c.Session["user.name"]
	if !ok {
		c.Flash.Error("Please login first")
		return c.Redirect(routes.App.Login())
	}
	var user model.User
	model.DB.Where("username = ?", username).First(&user)
	return c.Render(user)
}

func (c App) DoModifyUser(oldpassword, password, verifyPassword string) revel.Result {
	username, ok := c.Session["user.name"]
	if !ok {
		c.Flash.Error("Please login first")
		return c.Redirect(routes.App.Login())
	}
	if password == "" || password != verifyPassword {
		c.Flash.Error("Please check your new password")
		return c.Redirect(routes.App.ModifyUser())
	}
	var user model.User
	model.DB.Where("username = ?", username).First(&user)
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldpassword))
	if err != nil {
		c.Flash.Error("incorrect password")
		return c.Redirect(routes.App.ModifyUser())
	}
	hashed, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	user.Password = string(hashed)
	if err := model.DB.Save(user).Error; err != nil {
		c.Flash.Error("%v", err)
		c.FlashParams()
		return c.Redirect(routes.App.ModifyUser())
	}
	c.Flash.Success("Modify User Success")
	return c.Redirect(routes.App.Index())
}

func (c App) Logout() revel.Result {
	for k := range c.Session {
		delete(c.Session, k)
	}
	return c.Redirect(routes.App.Index())
}

func (c App) DoLogin(username, password string) revel.Result {
	_, pok := c.Session["user.name"]
	if pok {
		c.Flash.Error("You already logged in.")
		return c.Redirect(routes.App.Index())
	}
	var user model.User
	ret := model.DB.Where("username = ?", username).First(&user)
	if !ret.RecordNotFound() {
		err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
		if err == nil {
			c.Session["user.name"] = user.Username
			c.Session["user.id"] = fmt.Sprintf("%v", user.ID)
			c.Session["user.privilege"] = user.Privilege
			return c.Redirect(routes.App.Index())
		}
	}
	c.Flash.Error("Username or password incorrect.")
	c.FlashParams()
	return c.Redirect(routes.App.Login())
}

func (c App) DoRegister(user model.User, verifyPassword string) revel.Result {
	_, pok := c.Session["user.name"]
	if pok {
		c.Flash.Error("You already logged in.")
		return c.Redirect(routes.App.Index())
	}
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
	c.Session["user.id"] = fmt.Sprintf("%v", user.ID)
	c.Session["user.privilege"] = user.Privilege
	c.Flash.Success("Welcome")
	return c.Redirect(routes.App.Index())
}

func (c App) BatchRegister(usernames string) revel.Result {
	privilege, pok := c.Session["user.privilege"]
	if !pok || privilege != "admin" {
		c.Flash.Error("Permission Denied.")
		return c.Redirect(routes.App.Index())
	}
	const PASSWORD = "123456"
	l := strings.Split(usernames, "\n")
	tx := model.DB.Begin()
	for _, v := range l {
		user := model.User{
			Username:  strings.TrimSpace(v),
			Password:  PASSWORD,
			Privilege: "user",
		}
		hashed, _ := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		user.Password = string(hashed)
		tx.Create(&user)
	}
	if err := tx.Commit().Error; err != nil {
		c.Flash.Error("%v", err)
		c.FlashParams()
		return c.Redirect(routes.App.Management())
	}

	c.Flash.Success("Batch register success. All passwords are `%s`", PASSWORD)
	return c.Redirect(routes.App.Management())
}

func (c App) Problem(shortname string) revel.Result {
	var prob model.Problem
	p := model.DB.Where("short_name = ?", shortname).First(&prob)
	if p.RecordNotFound() {
		return c.NotFound("Problem %s Not Found", shortname)
	}

	privilege, pok := c.Session["user.privilege"]
	if time.Now().Before(prob.SecretBefore) && (!pok || privilege != "admin") {
		c.Flash.Error("The problem is still a secret.")
		return c.Redirect(routes.App.Index())
	}

	type dataType struct {
		DataID string
		MyBest int
		Best   int
	}
	data := make([]dataType, prob.NumTestcase+1)

	// get my best
	type mybestType struct {
		TestcaseID uint
		Max        int
	}
	var mybestRows []mybestType
	username := c.Session["user.name"]
	model.DB.Table("submits").
		Select("testcase_id, MAX(score)").
		Where("short_name = ? AND username = ?", prob.ShortName, username).
		Group("username, short_name, testcase_id").
		Scan(&mybestRows)
	for _, v := range mybestRows {
		data[v.TestcaseID].MyBest = v.Max
	}

	// get best of all
	type bestType struct {
		TestcaseID uint
		Max        int
	}
	var bestRows []bestType
	model.DB.Table("submits").
		Select("testcase_id, MAX(score)").
		Where("short_name = ?", prob.ShortName).
		Group("short_name, testcase_id").
		Scan(&bestRows)
	for _, v := range bestRows {
		data[v.TestcaseID].Best = v.Max
	}

	// set sum
	data[0].DataID = "sum"
	for i := 1; i <= prob.NumTestcase; i++ {
		data[i].DataID = strconv.Itoa(i)
		data[0].Best += data[i].Best
		data[0].MyBest += data[i].MyBest
	}

	return c.Render(prob, data)
}

func (c App) DownloadInput(shortname string, tid int) revel.Result {
	username, ok := c.Session["user.name"]
	if !ok {
		c.Flash.Error("Please Login First")
		return c.Redirect(routes.App.Login())
	}

	var prob model.Problem
	retp := model.DB.Where("short_name = ?", shortname).First(&prob)
	if retp.RecordNotFound() {
		return c.NotFound("Problem %s Not Found", shortname)
	}

	privilege, pok := c.Session["user.privilege"]
	if time.Now().Before(prob.SecretBefore) && (!pok || privilege != "admin") {
		c.Flash.Error("The problem is still a secret.")
		return c.Redirect(routes.App.Index())
	}

	var input model.ProblemInput
	ret := model.DB.Where("username = ? AND short_name = ? AND testcase_ID = ?",
		username, shortname, tid).First(&input)
	if ret.RecordNotFound() {
		c.Flash.Error("Cannot find data.")
		return c.Redirect(routes.App.Problem(shortname))
	}

	p := path.Join(SubmitBest.ROOT_USER_INPUT, input.InputFile)
	f, err := os.Open(p)
	if err != nil {
		c.Flash.Error("%v", err)
		return c.Redirect(routes.App.Problem(shortname))
	}
	d := fmt.Sprintf("attachment; filename=\"%d.in\"", tid)
	return c.RenderFile(f, revel.ContentDisposition(d))
}

func (c App) DownloadSolutionInput(sid int, filename string) revel.Result {
	var s model.Submit
	ret := model.DB.Where("id=?", sid).First(&s)
	if ret.RecordNotFound() {
		c.Flash.Error("Cannot find solution.")
		return c.Redirect(routes.App.Status())
	}

	username, pok := c.Session["user.name"]
	privilege, _ := c.Session["user.privilege"]
	if !pok || (username != s.Username && privilege != "admin") {
		c.Flash.Error("Permission Denied")
		return c.Redirect(routes.App.Status())
	}

	p := path.Join(SubmitBest.ROOT_USER_INPUT, s.InputFile)
	f, err := os.Open(p)
	if err != nil {
		c.Flash.Error("%v", err)
		return c.Redirect(routes.App.Status())
	}
	d := fmt.Sprintf("attachment; filename=\"%s\"", filename)
	return c.RenderFile(f, revel.ContentDisposition(d))
}

func (c App) DownloadSolutionAnswer(sid int, filename string) revel.Result {
	var s model.Submit
	ret := model.DB.Where("id=?", sid).First(&s)
	if ret.RecordNotFound() {
		c.Flash.Error("Cannot find solution.")
		return c.Redirect(routes.App.Status())
	}

	username, pok := c.Session["user.name"]
	privilege, _ := c.Session["user.privilege"]
	if !pok || (username != s.Username && privilege != "admin") {
		c.Flash.Error("Permission Denied")
		return c.Redirect(routes.App.Status())
	}

	p := path.Join(SubmitBest.ROOT_USER_ANSWER, s.AnswerFile)
	f, err := os.Open(p)
	if err != nil {
		c.Flash.Error("%v", err)
		return c.Redirect(routes.App.Status())
	}
	d := fmt.Sprintf("attachment; filename=\"%s\"", filename)
	return c.RenderFile(f, revel.ContentDisposition(d))
}

func (c App) DownloadSolution(sid int, filename string) revel.Result {
	var s model.Submit
	ret := model.DB.Where("id=?", sid).First(&s)
	if ret.RecordNotFound() {
		c.Flash.Error("Cannot find solution.")
		return c.Redirect(routes.App.Status())
	}

	username, pok := c.Session["user.name"]
	privilege, _ := c.Session["user.privilege"]
	if !pok || (username != s.Username && privilege != "admin") {
		c.Flash.Error("Permission Denied")
		return c.Redirect(routes.App.Status())
	}

	p := path.Join(SubmitBest.ROOT_USER_SOLUTION, s.SolutionFile)
	f, err := os.Open(p)
	if err != nil {
		c.Flash.Error("%v", err)
		return c.Redirect(routes.App.Status())
	}
	d := fmt.Sprintf("attachment; filename=\"%s\"", filename)
	return c.RenderFile(f, revel.ContentDisposition(d))
}

func (c App) Management() revel.Result {
	privilege, pok := c.Session["user.privilege"]
	if !pok || privilege != "admin" {
		c.Flash.Error("Permission Denied")
		return c.Redirect(routes.App.Index())
	}
	return c.Render()
}

func (c App) RefreshProblem(shortname string) revel.Result {
	privilege, pok := c.Session["user.privilege"]
	if !pok || privilege != "admin" {
		c.Flash.Error("Permission Denied")
		return c.Redirect(routes.App.Index())
	}

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
	privilege, pok := c.Session["user.privilege"]
	if !pok || privilege != "admin" {
		c.Flash.Error("Permission Denied")
		return c.Redirect(routes.App.Index())
	}

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
		fmt.Printf("    making data for (%s, %s, %d)\n", a.username, a.shortname, a.dataID)
		ret := model.DB.Where("username = ? AND short_name = ? AND testcase_id = ?",
			a.username, a.shortname, a.dataID).First(&old)
		if !ret.RecordNotFound() && !a.reload {
			continue
		}

		filename := path.Join(a.username, a.shortname, sbest.RandString(16))
		os.MkdirAll(path.Join(SubmitBest.ROOT_USER_INPUT, a.username, a.shortname), 0755)
		realpath := path.Join(SubmitBest.ROOT_USER_INPUT, filename)
		sbest.MakeData(a.shortname, a.username, a.dataID, realpath)

		if ret.RecordNotFound() {
			now := model.ProblemInput{
				Username:   a.username,
				ShortName:  a.shortname,
				TestcaseID: a.dataID,
				InputFile:  filename,
			}
			err = model.DB.Create(&now).Error
		} else {
			old.InputFile = filename
			err = model.DB.Save(&old).Error
		}
		if err != nil {
			log.Fatal(err)
		}
	}
	fmt.Printf("    all %d data is made.\n", len(arg))
	return nil
}

func (c App) MakeDataAll(shortname string, reload string) revel.Result {
	privilege, pok := c.Session["user.privilege"]
	if !pok || privilege != "admin" {
		c.Flash.Error("Permission Denied")
		return c.Redirect(routes.App.Index())
	}

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

	err := makeData(args)
	if err != nil {
		c.Flash.Error("%v", err)
		c.FlashParams()
		return c.Redirect(routes.App.Management())
	}

	c.Flash.Success("Successfully make data for problem %s for all users", shortname)
	return c.Redirect(routes.App.Problem(shortname))
}

func (c App) Submit() revel.Result {
	_, ok := c.Session["user.name"]
	if !ok {
		c.Flash.Error("You must login first!")
		c.FlashParams()
		return c.Redirect(routes.App.Login())
	}

	testcaseid := c.Params.Get("testcaseid")
	shortname := c.Params.Get("shortname")
	if testcaseid == "" {
		testcaseid = c.Flash.Data["testcaseid"]
	}
	if shortname == "" {
		shortname = c.Flash.Data["shortname"]
	}
	return c.Render(testcaseid, shortname)
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
		Username:         username,
		ShortName:        shortname,
		TestcaseID:       testcaseID,
		InputFile:        input.InputFile,
		SolutionFile:     pSolution,
		SolutionFileName: solution.Filename,
		AnswerFile:       pAnswer,
	}
	if err := model.DB.Create(&submit).Error; err != nil {
		c.Flash.Error("%v", err)
		c.FlashParams()
		return c.Redirect(routes.App.Submit())
	}

	return c.Redirect(routes.App.Solution(submit.ID))
}

func (c App) Status() revel.Result {
	var table []model.Submit
	username := c.Params.Get("username")
	shortname := c.Params.Get("shortname")
	testcaseid := c.Params.Get("testcaseid")
	start := c.Params.Get("start")
	fmt.Printf("%s, %s, %s\n", username, shortname, testcaseid)
	db := model.DB
	if username != "" {
		db = db.Where("username = ?", username)
	}
	if shortname != "" {
		db = db.Where("short_name = ?", shortname)
	}
	if testcaseid != "" {
		id, _ := strconv.Atoi(testcaseid)
		db = db.Where("testcase_id = ?", id)
	}
	if start != "" {
		s, _ := strconv.Atoi(start)
		db = db.Where("id <= ?", s)
	}
	db.Limit("25").Order("id DESC").Find(&table)
	return c.Render(table, username, shortname, testcaseid, start)
}

func (c App) Solution(id uint) revel.Result {
	var submit model.Submit
	p := model.DB.Where("id = ?", id).First(&submit)
	if p.RecordNotFound() {
		return c.NotFound("Solution %s Not Found", id)
	}

	username, pok := c.Session["user.name"]
	privilege, _ := c.Session["user.privilege"]
	if !pok || (username != submit.Username && privilege != "admin") {
		c.Flash.Error("Permission Denied")
		return c.Redirect(routes.App.Status())
	}

	return c.Render(submit)
}

func (c App) Contests() revel.Result {
	var cs []model.Contest
	model.DB.Order("id DESC").Find(&cs)
	type ct struct {
		ID                       uint
		Title, StartAt, FinishAt string
	}
	var contests []ct
	for _, c := range cs {
		contests = append(contests, ct{
			ID:       c.ID,
			Title:    c.Title,
			StartAt:  c.StartAt.Format(SubmitBest.TimeFormat),
			FinishAt: c.FinishAt.Format(SubmitBest.TimeFormat),
		})
	}
	return c.Render(contests)
}

func (c App) Contest(id uint) revel.Result {
	var contest model.Contest
	var problems []model.Problem
	ret := model.DB.Where("id = ?", id).First(&contest)
	if ret.RecordNotFound() {
		return c.NotFound("Contest %d not found.", id)
	}
	model.DB.Model(&contest).Related(&problems, "Problems")
	startAt := contest.StartAt.Format(SubmitBest.TimeFormat)
	finishAt := contest.FinishAt.Format(SubmitBest.TimeFormat)
	return c.Render(contest, problems, startAt, finishAt)
}

type rect struct {
	Username   string
	ShortName  string
	TestcaseID int
	Max        int
	Count      int
}
type rowProblemType struct {
	Sum      int
	Testcase []rect
}
type rowType struct {
	Score    int
	Username string
	Problem  []rowProblemType
}
type boardType []rowType

func (b boardType) Len() int {
	return len(b)
}
func (b boardType) Less(i, j int) bool {
	return b[i].Score > b[j].Score
}
func (b boardType) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (c App) Board(id uint) revel.Result {
	var contest model.Contest
	var problems []model.Problem
	ret := model.DB.Where("id = ?", id).First(&contest)
	if ret.RecordNotFound() {
		return c.NotFound("Contest %d not found.", id)
	}
	model.DB.Model(&contest).Related(&problems, "Problems")
	problemMap := make(map[string]*model.Problem)
	var problemNames []string
	for _, p := range problems {
		problemMap[p.ShortName] = &p
		problemNames = append(problemNames, p.ShortName)
	}

	var recs []rect
	model.DB.Table("submits").
		Select("username, short_name, testcase_id, MAX(score), COUNT(*)").
		Where("? <= created_at AND created_at < ? AND short_name IN (?)", contest.StartAt, contest.FinishAt, problemNames).
		Group("username, short_name, testcase_id").
		Scan(&recs)
	type pt struct {
		Sum int
		T   []rect
	}
	type rowt struct {
		Sum int
		M   map[string]*pt
	}
	rowMap := make(map[string]rowt)
	for _, r := range recs {
		row, ok := rowMap[r.Username]
		if !ok {
			row = rowt{0, make(map[string]*pt)}
			rowMap[r.Username] = row
		}
		prob, ok := row.M[r.ShortName]
		if !ok {
			t := problemMap[r.ShortName].NumTestcase
			prob = &pt{0, make([]rect, t)}
			row.M[r.ShortName] = prob
		}
		prob.Sum += r.Max
		prob.T[r.TestcaseID-1] = r
	}

	var board boardType
	for username, row := range rowMap {
		r := rowType{0, username, make([]rowProblemType, len(problems))}
		for i, prob := range problems {
			r.Score += row.M[prob.ShortName].Sum
			r.Problem[i].Sum = row.M[prob.ShortName].Sum
			r.Problem[i].Testcase = row.M[prob.ShortName].T
		}
		board = append(board, r)
	}
	sort.Sort(board)

	return c.Render(contest, problems, board)
}
