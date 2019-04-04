package sites

import (
	"github.com/ecletus/core"
	"github.com/ecletus/plug"
	"github.com/ecletus/router"
	"github.com/moisespsena-go/pluggable"
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
		prefix := Router.Server().Config.Prefix
		if prefix == "" {
			prefix = "/"
		}
		Handler.Log(prefix)
		Router.Handler = Handler
	})
}
