package app

import (
	"github.com/go-chi/chi/v5"
)

func (app *CortezaDiscoveryApp) MountHttpRoutes(r chi.Router) {
	//var (
	//	ho = app.Opt.HTTPServer
	//)

	//func() {
	//	//if !ho.ApiEnabled {
	//	//	app.Log.Info("JSON REST API disabled")
	//	//	return
	//	//}
	//
	//	r.Route(options.CleanBase(ho.ApiBaseUrl), func(r chi.Router) {
	//		var fullpathAPI = "/" + strings.TrimPrefix(options.CleanBase(ho.BaseUrl, ho.ApiBaseUrl), "/")
	//
	//		app.Log.Info(
	//			"JSON REST API enabled",
	//			zap.String("baseUrl", fullpathAPI),
	//			//zap.String("baseUrl", ho.BaseUrl),
	//		)
	//
	//		fmt.Println("app.Opt.Searcher.Enabled: ", app.Opt.Searcher.Enabled)
	//		if app.Opt.Searcher.Enabled {
	//			r.Route("/", searcherRest.MountRoutes())
	//		}
	//	})
	//}()

	//func() {
	//	if app.Opt.Searcher.Enabled {
	//		r.Route("/one", searcherRest.MountRoutes())
	//	}
	//
	//	//r.Handle("/.well-known/openid-configuration", app.AuthService.WellKnownOpenIDConfiguration())
	//}()

}
