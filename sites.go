package sites

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ecletus/core"
	"github.com/moisespsena-go/default-logger"
	"github.com/moisespsena-go/path-helpers"
	"github.com/moisespsena-go/xroute"
)

var log = defaultlogger.NewLogger(path_helpers.GetCalledDir())

type MiddlewareHandler func(context *core.Context, next func(context *core.Context))

type SitesRouter struct {
	ContextFactory *core.ContextFactory
	DefaultDomain  string
	DefaultSite    string
	Alone          bool
	ByDomain       bool
	Sites          core.SitesReader
	DomainsMap     map[string]core.SiteInterface
	SiteHandler    xroute.ContextHandler
	HandleNotFound xroute.ContextHandler
	HandleIndex    xroute.ContextHandler
	Middlewares    *xroute.MiddlewaresStack
}

func (sr *SitesRouter) All() []core.SiteInterface {
	return sr.Sites.All()
}

func NewSites(contextFactory *core.ContextFactory) *SitesRouter {
	r := &SitesRouter{
		ContextFactory: contextFactory,
		Sites:          make(core.SitesReader),
		DomainsMap:     make(map[string]core.SiteInterface),
		Middlewares:    xroute.NewMiddlewaresStack(PKG+".Middlewares", true),
		HandleNotFound: xroute.HttpHandler(http.NotFoundHandler()),
	}
	r.HandleIndex = xroute.HttpHandler(r.DefaultIndexHandler)
	return r
}

func (sr *SitesRouter) DefaultIndexHandler(w http.ResponseWriter, r *http.Request, rctx *xroute.RouteContext) {
	if sr.DefaultSite != "" {
		if site := sr.Get(sr.DefaultSite); site != nil {
			ContextGetSiteHandler(rctx)(w, r, rctx, site)
		} else {
			sr.HandleNotFound.ServeHTTPContext(w, r, rctx)
		}
		return
	}

	if sr.HandleIndex != nil {
		sr.HandleIndex.ServeHTTPContext(w, r, rctx)
	} else {
		sr.HandleNotFound.ServeHTTPContext(w, r, rctx)
	}
}

// Use reigster a middleware to the router
func (r *SitesRouter) Use(middlewares ...*xroute.Middleware) {
	r.Middlewares.Add(middlewares, xroute.DUPLICATION_ABORT)
}

// GetMiddleware get registered middleware
func (r *SitesRouter) GetMiddleware(name string) *xroute.Middleware {
	return r.Middlewares.ByName[name]
}

func (sites *SitesRouter) Get(name string) core.SiteInterface {
	return sites.Sites[name]
}

func (sites *SitesRouter) GetByDomain(host string) (site core.SiteInterface) {
	// host:port
	parts := strings.SplitN(host, ":", 2)
	if len(parts) != 2 {
		return
	}

	site = sites.DomainsMap[host]

	if site != nil {
		return
	}

	domain, port := parts[0], parts[1]

	// all ports
	site = sites.DomainsMap[domain]

	if site != nil {
		return
	}

	// only port
	site = sites.DomainsMap[":"+port]

	if site != nil {
		return
	}

	// subdomain of default domain
	domainParts := strings.SplitN(domain, ".", 2)

	if len(domainParts) > 1 && domainParts[1] == sites.DefaultDomain {
		site = sites.Get(domainParts[0])
	}
	return
}

func (sites *SitesRouter) Each(cb func(core.SiteInterface) error) (err error) {
	return sites.Sites.Each(cb)
}

func SiteStorageName(siteName, storageName string) string {
	return siteName + ":" + siteName
}

func (sites *SitesRouter) Register(site core.SiteInterface) {
	sites.Sites[site.Name()] = site
	for _, domain := range site.Config().Domains {
		sites.DomainsMap[domain] = site
	}
}

func (sites *SitesRouter) CreateSitesIndex() *SitesIndex {
	return &SitesIndex{Router: sites, PageTitle: "Site chooser"}
}

type SitesIndex struct {
	Router       *SitesRouter
	PageTitle    string
	StatusCode   int
	URI          string
	ExcludeSites []string
	excludes     map[string]bool
	Handler      func(sites []core.SiteInterface, w http.ResponseWriter, r *http.Request)
}

func (si *SitesIndex) ServeHTTPContext(w http.ResponseWriter, r *http.Request, rctx *xroute.RouteContext) {
	if si.excludes == nil {
		si.excludes = make(map[string]bool)
		for _, name := range si.ExcludeSites {
			si.excludes[name] = true
		}
	}

	var sites []core.SiteInterface
	for _, site := range si.Router.Sites.Sorted() {
		if _, ok := si.excludes[site.Name()]; !ok {
			sites = append(sites, site)
		}
	}

	if si.Handler != nil {
		si.Handler(sites, w, r)
		return
	}

	pth := strings.TrimSuffix(r.RequestURI, "/")

	msg := `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>` + si.PageTitle + `</title>
</head>
<body>
<h1>` + si.PageTitle + `</h1>
<ul>
`

	for _, site := range sites {
		msg += fmt.Sprintf(`<li><a href="%v/%v/">%v</a></li>`, pth, site.Name(), site.Name())
	}

	msg += `
</ul>
</body>
</html>`

	stausCode := si.StatusCode
	if stausCode == 0 {
		stausCode = 200
	}
	w.WriteHeader(stausCode)
	w.Write([]byte(msg))
}
