package sites

import (
	"github.com/ecletus/core"
	"github.com/ecletus/plug"
	"github.com/ecletus/router"
)

type RouterPlugin struct {
	plug.EventDispatcher
	RouterKey, SitesRouterKey string
	Alone                     bool
}

func (p *RouterPlugin) RequireOptions() []string {
	return []string{p.RouterKey, p.SitesRouterKey}
}

func (p *RouterPlugin) OnRegister() {
	router.OnRoute(p, func(e *router.RouterEvent) {
		sitesRouter := e.Options().GetInterface(p.SitesRouterKey).(*SitesRouter)
		Router := e.Router
		var Handler *SitesHandler

		if p.Alone || sitesRouter.Register.Alone {
			Handler = sitesRouter.CreateAloneHandler()
		} else {
			Handler = sitesRouter.CreateHandler()
		}

		mux := Router.GetMux()
		sitesRouter.Register.OnAdd(func(site *core.Site) {
			if len(site.Middlewares) > 0 {
				site.SetHandler(site.Middlewares.Handler(mux))
			} else {
				site.SetHandler(mux)
			}
		})
		sitesRouter.Register.OnSiteDestroy(func(site *core.Site) {
			site.SetHandler(nil)
		})
		Router.Handler = Handler
	})
}
