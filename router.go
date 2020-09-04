package sites

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"sort"
	"strings"

	"github.com/moisespsena-go/middleware"

	"github.com/ecletus/core"
	defaultlogger "github.com/moisespsena-go/default-logger"
	path_helpers "github.com/moisespsena-go/path-helpers"
	"github.com/moisespsena-go/xroute"
)

var log = defaultlogger.GetOrCreateLogger(path_helpers.GetCalledDir())

type MiddlewareHandler func(context *core.Context, next func(context *core.Context))

type SitesRouter struct {
	Prefix                      string
	NotMountNames               bool
	RedirectSiteNotFoundToIndex bool
	ContextFactory              *core.ContextFactory
	DefaultDomain               string
	DefaultSite                 string
	Register                    *core.SitesRegister
	SiteHandler                 xroute.ContextHandler
	HandleNotFound              xroute.ContextHandler
	HandleIndex                 xroute.ContextHandler
	Middlewares                 *xroute.MiddlewaresStack
}

func NewSitesRouter(register *core.SitesRegister, contextFactory *core.ContextFactory) *SitesRouter {
	r := &SitesRouter{
		ContextFactory: contextFactory,
		Register:       register,
		Middlewares:    xroute.NewMiddlewaresStack(PKG+".Middlewares", true),
		HandleNotFound: xroute.HttpHandler(http.NotFoundHandler()),
	}
	r.HandleIndex = xroute.HttpHandler(r.DefaultIndexHandler)
	return r
}

func (this *SitesRouter) Init() {
	if !this.Register.Alone {
		this.Register.OnPathAdd(func(site *core.Site, pth string) {
			log.Infof("[%s] path: mounted on %s", site.Name(), path.Join("/", this.Prefix, pth))
		})
		this.Register.OnPathDel(func(site *core.Site, pth string) {
			log.Infof("[%s] path: ummounted from %s", site.Name(), path.Join("/", this.Prefix, pth))
		})
	}
	this.Register.OnHostAdd(func(site *core.Site, host string) {
		log.Infof("[%s] host: mounted on %s", site.Name(), host)
	})
	this.Register.OnHostDel(func(site *core.Site, host string) {
		log.Infof("[%s] host: ummounted from %s", site.Name(), host)
	})
	this.Register.OnAdd(func(site *core.Site) {
		log.Infof("[%s] added", site.Name())

		fmtr := site.RequestLogger("log/http")
		if fmtr == nil {
			fmtr = middleware.DefaultRequestLogFormatter
		}
		site.Middlewares.Add(xroute.NewMiddleware(func(next http.Handler) http.Handler {
			next = middleware.RequestLogger(fmtr)(next)
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				r = r.WithContext(context.WithValue(r.Context(), xroute.SkipRequestLogger, true))
				next.ServeHTTP(w, r)
			})
		}))
		site.Middlewares.Add(xroute.NewMiddleware(func(next http.Handler) http.Handler {
			next = middleware.Recoverer(fmtr)(next)
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				r = r.WithContext(context.WithValue(r.Context(), xroute.SkipErrorInterseption, true))
				next.ServeHTTP(w, r)
			})
		}))
	})
	this.Register.OnSiteDestroy(func(site *core.Site) {
		log.Infof("[%s] deleted", site.Name())
	})
	this.Register.OnPostAdd(func(site *core.Site) {
		if !this.NotMountNames {
			this.Register.AddPath(site.Name(), site.Name())
		}
	})
}

func (this *SitesRouter) DefaultIndexHandler(w http.ResponseWriter, r *http.Request, rctx *xroute.RouteContext) {
	if this.DefaultSite != "" {
		if site, ok := this.Register.Get(this.DefaultSite); ok {
			ContextGetSiteHandler(rctx)(w, r, rctx, site)
		} else {
			this.HandleNotFound.ServeHTTPContext(w, r, rctx)
		}
		return
	}

	this.HandleNotFound.ServeHTTPContext(w, r, rctx)
}

// Use reigster a middleware to the router
func (this *SitesRouter) Use(middlewares ...*xroute.Middleware) {
	this.Middlewares.Add(middlewares, xroute.DUPLICATION_ABORT)
}

// GetMiddleware get registered middleware
func (this *SitesRouter) GetMiddleware(name string) *xroute.Middleware {
	return this.Middlewares.ByName[name]
}

func (this *SitesRouter) GetByHost(host string) (site *core.Site) {
	// host:port
	parts := strings.SplitN(host, ":", 2)
	if len(parts) == 1 {
		var ok bool
		// by host
		if site, ok = this.Register.GetByHost(host); ok {
			return
		}
	}
	return
}
func (this *SitesRouter) CreateSitesIndex() *SitesIndex {
	return &SitesIndex{Router: this, PageTitle: "Site chooser"}
}

func SiteStorageName(siteName, storageName string) string {
	return siteName + ":" + siteName
}

type SitesIndex struct {
	Router       *SitesRouter
	PageTitle    string
	StatusCode   int
	URI          string
	ExcludeSites []string
	excludes     map[string]bool
	Handler      func(sites []*core.Site, w http.ResponseWriter, r *http.Request)
}

func (this *SitesIndex) ServeHTTPContext(w http.ResponseWriter, r *http.Request, rctx *xroute.RouteContext) {
	if this.excludes == nil {
		this.excludes = make(map[string]bool)
		for _, name := range this.ExcludeSites {
			this.excludes[name] = true
		}
	}

	var sites []*core.Site
	for _, site := range this.Router.Register.ByName.Sorted() {
		if _, ok := this.excludes[site.Name()]; !ok {
			sites = append(sites, site)
		}
	}

	if this.Handler != nil {
		this.Handler(sites, w, r)
		return
	}

	pth := strings.TrimSuffix(r.RequestURI, "/")

	msg := `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>` + this.PageTitle + `</title>
</head>
<body>
<h1>` + this.PageTitle + `</h1>
<ul>
`

	for _, site := range sites {
		msg += fmt.Sprintf(`<li><a href="%v/%v/">%v</a></li>`, pth, site.Name(), site.Name())
	}

	paths := this.Router.Register.ByPath.Keys()
	sort.Strings(paths)

	for _, sitePth := range paths {
		if site, ok := this.Router.Register.GetByPath(sitePth); ok {
			msg += fmt.Sprintf(`<li><a href="%v/%v/">%v</a></li>`, pth, sitePth, site.Name())
		}
	}

	msg += `
</ul>
</body>
</html>`

	stausCode := this.StatusCode
	if stausCode == 0 {
		stausCode = 200
	}
	w.WriteHeader(stausCode)
	w.Write([]byte(msg))
}
