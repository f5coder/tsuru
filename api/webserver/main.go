// +build ignore

package main

import (
	"."
	"flag"
	"github.com/bmizerany/pat"
	"github.com/timeredbull/tsuru/api/app"
	"github.com/timeredbull/tsuru/api/auth"
	"github.com/timeredbull/tsuru/config"
	"github.com/timeredbull/tsuru/api/service"
	"github.com/timeredbull/tsuru/db"
	"github.com/timeredbull/tsuru/log"
	stdlog "log"
	"log/syslog"
	"net/http"
)

func main() {
	var err error
	log.Target, err = syslog.NewLogger(syslog.LOG_INFO, stdlog.LstdFlags)
	if err != nil {
		panic(err)
	}
	configFile := flag.String("config", "/etc/tsuru/tsuru.conf", "tsuru config file")
	dry := flag.Bool("dry", false, "dry-run: does not start the server (for testing purpose)")
	flag.Parse()
	err = config.ReadConfigFile(*configFile)
	if err != nil {
		log.Panic(err.Error())
	}
	db.Session, err = db.Open(config.GetString("database:host"), config.GetString("database:name"))
	if err != nil {
		log.Panic(err.Error())
	}
	defer db.Session.Close()
	m := pat.New()

	m.Post("/services", webserver.Handler(service.CreateHandler))
	m.Get("/services", webserver.Handler(service.ServicesHandler))
	m.Get("/services/types", webserver.Handler(service.ServiceTypesHandler))
	m.Get("/services/:name", webserver.Handler(service.DeleteHandler))
	m.Post("/services/bind", webserver.Handler(service.BindHandler))
	m.Post("/services/unbind", webserver.Handler(service.UnbindHandler))

	m.Get("/apps/:name/delete", webserver.Handler(app.AppDelete))
	m.Get("/apps/:name/clone", webserver.Handler(app.CloneRepositoryHandler))
	m.Get("/apps/:name", webserver.Handler(app.AppInfo))
	m.Post("/apps/:name/application", webserver.Handler(app.Upload))
	m.Get("/apps", webserver.Handler(app.AppList))
	m.Post("/apps", webserver.Handler(app.CreateAppHandler))
	m.Put("/apps/:app/:team", webserver.AuthorizationRequiredHandler(app.GrantAccessToTeamHandler))
	m.Del("/apps/:app/:team", webserver.AuthorizationRequiredHandler(app.RevokeAccessFromTeamHandler))

	m.Post("/users", webserver.Handler(auth.CreateUser))
	m.Post("/users/:email/tokens", webserver.Handler(auth.Login))
	m.Get("/users/check-authorization", webserver.Handler(auth.CheckAuthorization))

	m.Post("/teams", webserver.AuthorizationRequiredHandler(auth.CreateTeam))
	m.Put("/teams/:team/:user", webserver.AuthorizationRequiredHandler(auth.AddUserToTeam))
	m.Del("/teams/:team/:user", webserver.AuthorizationRequiredHandler(auth.RemoveUserFromTeam))

	listen := config.GetString("listen")
	if !*dry {
		log.Fatal(http.ListenAndServe(listen, m))
	}
}