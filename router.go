package sites

import (
	"net/http"
	"strings"

	"github.com/aghape/core"
	"github.com/moisespsena/go-route"
)

func (sites *SitesRouter) MountTo(path string, rootMux *http.ServeMux) *SitesRouter {
	sites.Prefix = path
	rootMux.Handle(path, sites.Mux())
	return sites
}

func (sites *SitesRouter) Log(prefix string) {
	if prefix != "" {
		prefix = strings.TrimSuffix(prefix, "/") + "/"
	}
	if sites.Alone {
		sites.Each(func(site core.SiteInterface) bool {
			log.Infof("Site %q mounted on %v", site.Name(), prefix+sites.Prefix)
			return true
		})
	} else {
		sites.Each(func(site core.SiteInterface) bool {
			log.Infof("Site %q mounted on %v", site.Name(), prefix+sites.Prefix+site.Name())
			return true
		})
	}
}

type SitesHandler struct {
	Sites       *SitesRouter
	middlewares *route.MiddlewaresStack
	Alone       bool
}

func (r *SitesHandler) Log(prefix string) {
	r.Sites.Log(prefix)
}

func (mux *SitesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r, rctx := route.GetOrNewRouteContextForRequest(r)
	mux.ServeHTTPContext(w, r, rctx)
}

func (mux *SitesHandler) ServeHTTPContext(w http.ResponseWriter, r *http.Request, rctx *route.RouteContext) {
	var site core.SiteInterface
	ContextSetSites(rctx, mux.Sites.Sites)
	ContextSetSiteHandler(rctx, mux.SiteHandler)

	if mux.Sites.Alone {
		mux.Sites.Each(func(s core.SiteInterface) bool {
			site = s
			return false
		})
	} else if r.URL.Path == "/" && mux.Sites.DefaultSite != "" {
		url := *route.GetOriginalURL(r)
		url.Path = strings.TrimSuffix(url.Path, "/") + "/" + mux.Sites.DefaultSite + "/"
		us := url.String()
		http.Redirect(w, r, us, http.StatusSeeOther)
		return
	} else {
		path := r.URL.Path
		sites := mux.Sites

		if path == "/favicon.ico" {
			http.NotFound(w, r)
			return
		}

		site = sites.GetByDomain(r.Host)

		if site == nil {
			path = strings.TrimPrefix(path, sites.DefaultPrefix)
			parts := strings.SplitN(strings.TrimLeft(path, "/"), "/", 2)

			if len(parts) == 0 {
				if sites.HandleIndex != nil {
					sites.HandleIndex.ServeHTTPContext(w, r, rctx)
				} else if sites.HandleNotFound != nil {
					sites.HandleNotFound.ServeHTTPContext(w, r, rctx)
				}
				return
			}

			var ctx *core.Context

			if len(parts) == 1 {
				r, ctx = sites.ContextFactory.GetOrNewContextFromRequestPair(w, r)
				url := ctx.GenURL(parts[0]) + "/"
				if url == path {
					if sites.HandleIndex != nil {
						sites.HandleIndex.ServeHTTPContext(w, r, rctx)
					} else {
						sites.HandleNotFound.ServeHTTPContext(w, r, rctx)
					}
					return
				}
				http.Redirect(w, r, url, http.StatusPermanentRedirect)
				return
			}

			siteName := parts[0]

			site = sites.Get(siteName)
			if site != nil {
				r.URL.Path = strings.TrimPrefix(r.URL.Path, "/"+siteName)
				r = sites.ContextFactory.SetSkipPrefixToRequest(r, sites.DefaultPrefix+"/"+siteName)
			}
		}

		if site == nil {
			sites.HandleNotFound.ServeHTTPContext(w, r, rctx)
			return
		}
	}

	mux.SiteHandler(w, r, rctx, site)
}

func (mux *SitesHandler) SiteHandler(w http.ResponseWriter, r *http.Request, rctx *route.RouteContext, site core.SiteInterface) {
	ContextSetSite(rctx, site)
	chain := mux.middlewares.Items.Handler(route.NewContextHandler(func(w http.ResponseWriter, r *http.Request, rctx *route.RouteContext) {
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

type SiteHandler func(w http.ResponseWriter, r *http.Request, rctx *route.RouteContext, site core.SiteInterface)

func ContextSetSiteHandler(rctx *route.RouteContext, handler SiteHandler) {
	rctx.Data[PKG+".siteHandler"] = handler
}

func ContextGetSiteHandler(rctx *route.RouteContext) SiteHandler {
	return rctx.Data[PKG+".siteHandler"].(SiteHandler)
}

func ContextSetSite(rctx *route.RouteContext, site core.SiteInterface) {
	rctx.Data[PKG+".site"] = site
}

func ContextGetSite(rctx *route.RouteContext) core.SiteInterface {
	return rctx.Data[PKG+".site"].(core.SiteInterface)
}

func ContextSetSites(rctx *route.RouteContext, sites core.SitesReaderInterface) {
	rctx.Data[PKG+".sites"] = sites
}

func ContextGetSites(rctx *route.RouteContext) core.SitesReaderInterface {
	return rctx.Data[PKG+".sites"].(core.SitesReaderInterface)
}
