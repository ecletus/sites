package sites

import (
	"net/http"
	"strings"

	"github.com/ecletus/core"
	"github.com/moisespsena-go/httpu"
	"github.com/moisespsena-go/xroute"
)

func (sites *SitesRouter) MountTo(path string, rootMux *http.ServeMux) *SitesRouter {
	rootMux.Handle(path, sites.Mux())
	return sites
}

func (sites *SitesRouter) Log(prefix string) {
	if prefix != "" {
		prefix = strings.TrimSuffix(prefix, "/") + "/"
	}
	if sites.Alone {
		sites.Each(func(site core.SiteInterface) error {
			log.Infof("Site %q mounted on %v", site.Name(), prefix)
			return nil
		})
	} else {
		sites.Each(func(site core.SiteInterface) error {
			log.Infof("Site %q mounted on %v", site.Name(), prefix+site.Name())
			return nil
		})
	}
}

type SitesHandler struct {
	Sites       *SitesRouter
	middlewares *xroute.MiddlewaresStack
	Alone       bool
}

func (r *SitesHandler) Log(prefix string) {
	r.Sites.Log(prefix)
}

func (mux *SitesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r, rctx := xroute.GetOrNewRouteContextForRequest(r)
	mux.ServeHTTPContext(w, r, rctx)
}

func (mux *SitesHandler) ServeHTTPContext(w http.ResponseWriter, r *http.Request, rctx *xroute.RouteContext) {
	var site core.SiteInterface
	ContextSetSites(rctx, mux.Sites.Sites)
	ContextSetSiteHandler(rctx, mux.SiteHandler)

	if mux.Sites.Alone {
		mux.Sites.Each(func(s core.SiteInterface) error {
			site = s
			return core.StopSiteIteration
		})
	} else if path := r.URL.Path; path == "/" {
		if mux.Sites.DefaultSite != "" {
			http.Redirect(w, r, path+mux.Sites.DefaultSite+"/", http.StatusSeeOther)
		} else if mux.Sites.HandleIndex != nil {
			mux.Sites.HandleIndex.ServeHTTPContext(w, r, rctx)
		} else {
			mux.Sites.HandleNotFound.ServeHTTPContext(w, r, rctx)
		}
		return
	} else if path == "/favicon.ico" {
		http.NotFound(w, r)
		return
	} else {
		sites := mux.Sites
		site = sites.GetByDomain(r.Host)

		if site == nil {
			parts := strings.SplitN(strings.Trim(path, "/"), "/", 2)
			siteName := parts[0]
			site = sites.Get(siteName)
			if site != nil {
				r.URL.Path = strings.TrimPrefix(r.URL.Path, "/"+siteName)
				r = httpu.PushPrefixR(r, siteName)
			}
		}

		if site == nil {
			sites.HandleNotFound.ServeHTTPContext(w, r, rctx)
			return
		}
	}

	mux.SiteHandler(w, r, rctx, site)
}

func (mux *SitesHandler) SiteHandler(w http.ResponseWriter, r *http.Request, rctx *xroute.RouteContext, site core.SiteInterface) {
	ContextSetSite(rctx, site)
	chain := mux.middlewares.Items.Handler(xroute.NewContextHandler(func(w http.ResponseWriter, r *http.Request, rctx *xroute.RouteContext) {
		site.ServeHTTPContext(w, r, rctx)
	}))

	chain.ServeHTTPContext(w, r, rctx)
}

func (sites *SitesRouter) CreateHandler() *SitesHandler {
	return &SitesHandler{sites, sites.Middlewares.Build(), false}
}

func (sites *SitesRouter) CreateAloneHandler() *SitesHandler {
	h := sites.CreateHandler()
	h.Alone = true
	return h
}

func (sites *SitesRouter) Mux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/", sites.CreateHandler())
	return mux
}

type SiteHandler func(w http.ResponseWriter, r *http.Request, rctx *xroute.RouteContext, site core.SiteInterface)

func ContextSetSiteHandler(rctx *xroute.RouteContext, handler SiteHandler) {
	rctx.Data[PKG+".siteHandler"] = handler
}

func ContextGetSiteHandler(rctx *xroute.RouteContext) SiteHandler {
	return rctx.Data[PKG+".siteHandler"].(SiteHandler)
}

func ContextSetSite(rctx *xroute.RouteContext, site core.SiteInterface) {
	rctx.Data[PKG+".site"] = site
}

func ContextGetSite(rctx *xroute.RouteContext) core.SiteInterface {
	return rctx.Data[PKG+".site"].(core.SiteInterface)
}

func ContextSetSites(rctx *xroute.RouteContext, sites core.SitesReaderInterface) {
	rctx.Data[PKG+".sites"] = sites
}

func ContextGetSites(rctx *xroute.RouteContext) core.SitesReaderInterface {
	return rctx.Data[PKG+".sites"].(core.SitesReaderInterface)
}
