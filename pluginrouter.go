package sites

import (
	"github.com/aghape/core"
	"github.com/aghape/plug"
	"github.com/aghape/router"
	"github.com/moisespsena/go-pluggable"
)

type RouterPlugin struct {
	plug.EventDispatcher
	RouterKey, SitesRouterKey string
	Alone                     bool
}

func (p *RouterPlugin) RequireOptions() []string {
	return []string{p.SitesRouterKey}
}

func (p *RouterPlugin) OnRegister(dis pluggable.PluginEventDispatcherInterface) {
	router.OnRoute(p, func(e *router.RouterEvent) {
		Sites := e.Options().GetInterface(p.SitesRouterKey).(*SitesRouter)
		Router := e.Router
		var Handler *SitesHandler

		if p.Alone || Sites.Alone {
			Handler = Sites.CreateAloneHandler()
		} else {
			Handler = Sites.CreateHandler()
		}

		mux := Router.GetRootMux()
		
		Sites.Each(func(site core.SiteInterface) error {
			site.(*core.Site).Handler = mux
			return nil
		})
		Handler.Log(Sites.DefaultPrefix)
		Router.Handler = Handler
	})
}
