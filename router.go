package sites

import (
	"net/http"
	"strings"

	"github.com/moisespsena/go-route"
	"github.com/aghape/core"
)

func (sites *SitesRouter) MountTo(path string, rootMux *http.ServeMux) *SitesRouter {
	sites.Prefix = path
	rootMux.Handle(path, sites.Mux())
	return sites
}

func (sites *SitesRouter) Log(prefix string) {
	if prefix != "" {
		prefix += "/"
	}
	sites.Each(func(site core.SiteInterface) bool {
		log.Info("New site:", site.Name(), "mounted on", prefix, sites.Prefix, site.Name())
		return true
	})
}

type SitesHandler struct {
	Sites       *SitesRouter
	middlewares *route.MiddlewaresStack
	Sinple      bool
}

func (r *SitesHandler) Log(prefix string) {
	r.Sites.Log(prefix)
}

func (mux *SitesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux.ServeHTTPContext(w, r, nil)
}

func (mux *SitesHandler) ServeHTTPContext(w http.ResponseWriter, r *http.Request, rctx *route.RouteContext) {
	var site core.SiteInterface

	if !mux.Sinple {
		path := r.URL.Path
		if path == "/favicon.ico" {
			http.NotFound(w, r)
			return
		}

		sites := mux.Sites
		site = sites.GetByDomain(r.Host)

		if site == nil {
			path = strings.TrimPrefix(path, sites.DefaultPrefix)
			parts := strings.SplitN(strings.TrimLeft(path, "/"), "/", 2)

			if len(parts) == 0 {
				if sites.HandleIndex != nil {
					sites.HandleIndex.ServeHTTP(w, r)
				} else if sites.HandleNotFound != nil {
					sites.HandleNotFound.ServeHTTP(w, r)
				} else {
					http.NotFound(w, r)
				}
				return
			}

			var ctx *core.Context

			if len(parts) == 1 {
				r, ctx = sites.ContextFactory.GetOrNewContextFromRequestPair(w, r)
				url := ctx.GenURL(parts[0]) + "/"
				if url == path {
					if sites.HandleIndex != nil {
						sites.HandleIndex.ServeHTTP(w, r)
					} else if sites.HandleNotFound != nil {
						sites.HandleNotFound.ServeHTTP(w, r)
					} else {
						http.NotFound(w, r)
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
			if sites.HandleNotFound != nil {
				sites.HandleNotFound.ServeHTTP(w, r)
			} else {
				http.NotFound(w, r)
			}
			return
		}
	} else {
		mux.Sites.Each(func(s core.SiteInterface) bool {
			site = s
			return false
		})
	}

	chain := mux.middlewares.Items.Handler(route.NewContextHandler(func(w http.ResponseWriter, r *http.Request, rctx *route.RouteContext) {
		site.ServeHTTPContext(w, r, rctx)
	}))

	chain.ServeHTTPContext(w, r, rctx)
}

func (sites *SitesRouter) CreateHandler() *SitesHandler {
	return &SitesHandler{sites, sites.Middlewares.Build(), false}
}

func (sites *SitesRouter) CreateSimpleHandler() *SitesHandler {
	h := sites.CreateHandler()
	h.Sinple = true
	return h
}

func (sites *SitesRouter) Mux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/", sites.CreateHandler())
	return mux
}
