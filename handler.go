package sites

import (
	"context"
	"github.com/ecletus/core/utils/url"
	"net/http"
	"strings"

	"github.com/moisespsena-go/httpu"

	"github.com/ecletus/core"
	"github.com/moisespsena-go/xroute"
)

type rootPath uint8

const RootPathKey rootPath = iota

func RootPath(r *http.Request) string {
	if v := r.Context().Value(RootPathKey); v != nil {
		return v.(string)
	}
	return "/"
}

type SitesHandler struct {
	Sites       *SitesRouter
	middlewares *xroute.MiddlewaresStack
	Alone       bool
}

func (this *SitesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r, rctx := xroute.GetOrNewRouteContextForRequest(r)
	this.ServeHTTPContext(w, r, rctx)
}

func (this *SitesHandler) ServeHTTPContext(w http.ResponseWriter, r *http.Request, rctx *xroute.RouteContext) {
	if !this.Serve(w, r) {
		if this.Sites.HandleNotFound != nil {
			this.Sites.HandleNotFound.ServeHTTPContext(w, r, rctx)
		} else {
			http.NotFound(w, r)
		}
	}
}

func (this *SitesHandler) Serve(w http.ResponseWriter, r *http.Request) (ok bool) {
	r, rctx := xroute.GetOrNewRouteContextForRequest(r)
	return this.ServeContext(w, r, rctx)
}

func (this *SitesHandler) ServeContext(w http.ResponseWriter, r *http.Request, rctx *xroute.RouteContext) (ok bool) {
	var site *core.Site

	ContextSetSiteHandler(rctx, this.SiteHandler)

	r = r.WithContext(context.WithValue(r.Context(), RootPathKey, this.Sites.Prefix))

	if this.Sites.Register.Alone {
		if site = this.Sites.Register.Site(); site != nil {
			this.SiteHandler(w, r, rctx, site)
			return true
		}
		return
	} else if path := r.URL.Path; path == "/" {
		if this.Sites.DefaultSite != "" {
			http.Redirect(w, r, path+this.Sites.DefaultSite+"/", http.StatusSeeOther)
			return true
		} else if this.Sites.HandleIndex != nil {
			this.Sites.HandleIndex.ServeHTTPContext(w, r, rctx)
			return true
		}
		return
	} else if path == "/favicon.ico" {
		return
	}

	if site == nil {
		sites := this.Sites
		site = sites.GetByHost(r.Host)
		parts := strings.SplitN(strings.Trim(r.RequestURI, "/"), "/", 2)
		if len(parts) == 1 && !strings.HasSuffix(r.URL.Path, "/") {
			newUrl := url.MustJoinURL(r.RequestURI, "/")
			http.Redirect(w, r, newUrl, http.StatusPermanentRedirect)
			return true
		}
		sitePath := parts[0]
		if site, ok = sites.Register.GetByPath(sitePath); !ok {
			site, ok = sites.Register.ByName.Get(sitePath)
		}
		if ok {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, "/"+sitePath)
			r = httpu.PushPrefixR(r, sitePath)
		} else if this.Sites.RedirectSiteNotFoundToIndex {
			this.Sites.HandleIndex.ServeHTTPContext(w, r, rctx)
			return true
		}
	}

	if ok {
		this.SiteHandler(w, r, rctx, site)
		return true
	}

	return false
}

func (this *SitesHandler) SiteHandler(w http.ResponseWriter, r *http.Request, rctx *xroute.RouteContext, site *core.Site) {
	ContextSetSite(rctx, site)
	chain := this.middlewares.Items.Handler(xroute.NewContextHandler(func(w http.ResponseWriter, r *http.Request, rctx *xroute.RouteContext) {
		site.ServeHTTPContext(w, r, rctx)
	}))
	chain.ServeHTTPContext(w, r, rctx)
}

func (this *SitesRouter) CreateHandler() *SitesHandler {
	return &SitesHandler{this, this.Middlewares.Build(), false}
}

func (this *SitesRouter) CreateAloneHandler() *SitesHandler {
	h := this.CreateHandler()
	h.Alone = true
	return h
}

func (this *SitesRouter) Mux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/", this.CreateHandler())
	return mux
}

type SiteHandler func(w http.ResponseWriter, r *http.Request, rctx *xroute.RouteContext, site *core.Site)

func ContextSetSiteHandler(rctx *xroute.RouteContext, handler SiteHandler) {
	rctx.Data[PKG+".siteHandler"] = handler
}

func ContextGetSiteHandler(rctx *xroute.RouteContext) SiteHandler {
	return rctx.Data[PKG+".siteHandler"].(SiteHandler)
}

func ContextSetSite(rctx *xroute.RouteContext, site *core.Site) {
	rctx.Data[PKG+".site"] = site
}

func ContextGetSite(rctx *xroute.RouteContext) *core.Site {
	return rctx.Data[PKG+".site"].(*core.Site)
}