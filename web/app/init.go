package app

import (
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/abcdabcd987/SubmitBest"
	"github.com/abcdabcd987/SubmitBest/model"
	"github.com/abcdabcd987/SubmitBest/sbest"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/revel/revel"
)

func init() {
	// Filters is the default set of global filters.
	revel.Filters = []revel.Filter{
		revel.PanicFilter,             // Recover from panics and display an error page instead.
		revel.RouterFilter,            // Use the routing table to select the right Action
		revel.FilterConfiguringFilter, // A hook for adding or removing per-Action filters.
		revel.ParamsFilter,            // Parse parameters into Controller.Params.
		revel.SessionFilter,           // Restore and write the session cookie.
		revel.FlashFilter,             // Restore and write the flash cookie.
		revel.ValidationFilter,        // Restore kept validation errors and save new ones from cookie.
		revel.I18nFilter,              // Resolve the requested language
		HeaderFilter,                  // Add some security based headers
		revel.InterceptorFilter,       // Run interceptors around the action.
		revel.CompressFilter,          // Compress the result.
		revel.ActionInvoker,           // Invoke the action.
	}

	revel.TemplateFuncs["myrange"] = func(a, b int) []int {
		x := make([]int, b-a+1)
		for i := a; i <= b; i++ {
			x[i-a] = i
		}
		return x
	}

	revel.TemplateFuncs["add"] = func(a, b int) int {
		return a + b
	}

	// register startup functions with OnAppStart
	// ( order dependent )
	revel.OnAppStart(initRand)
	revel.OnAppStart(initDB)
	revel.OnAppStart(initFileServer)
	revel.OnAppStart(initTaskServer)
	// revel.OnAppStart(FillCache)
}

func initRand() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func initDB() {
	host := SubmitBest.DB_HOST
	user := SubmitBest.DB_USER
	pass := SubmitBest.DB_PASS
	name := SubmitBest.DB_NAME
	model.InitDB(host, user, pass, name)
}

func initFileServer() {
	go func() {
		fs := http.FileServer(http.Dir(SubmitBest.ROOT))
		http.Handle("/", fs)
		log.Fatal(http.ListenAndServe(SubmitBest.FILESERVER_ADDR, fs))
	}()
}

func initTaskServer() {
	sbest.RunTaskServer(SubmitBest.TASKSERVER_ADDR)
}

// TODO turn this into revel.HeaderFilter
// should probably also have a filter for CSRF
// not sure if it can go in the same filter or not
var HeaderFilter = func(c *revel.Controller, fc []revel.Filter) {
	// Add some common security headers
	c.Response.Out.Header().Add("X-Frame-Options", "SAMEORIGIN")
	c.Response.Out.Header().Add("X-XSS-Protection", "1; mode=block")
	c.Response.Out.Header().Add("X-Content-Type-Options", "nosniff")

	fc[0](c, fc[1:]) // Execute the next filter stage.
}
