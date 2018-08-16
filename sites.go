package sites

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/moisespsena/go-default-logger"
	"github.com/moisespsena/go-path-helpers"
	"github.com/moisespsena/go-route"
	"github.com/aghape/aghape"
)

var log = defaultlogger.NewLogger(path_helpers.GetCalledDir())

type MiddlewareHandler func(context *qor.Context, next func(context *qor.Context))

type SitesRouter struct {
	ContextFactory *qor.ContextFactory
	DefaultDomain  string
	DefaultPrefix  string
	ByDomain       bool
	Sites          qor.SitesReader
	DomainsMap     map[string]qor.SiteInterface
	SiteHandler    route.ContextHandler
	HandleNotFound http.Handler
	HandleIndex    http.Handler
	Prefix         string
	Middlewares    *route.MiddlewaresStack
}

func NewSites(contextFactory *qor.ContextFactory) *SitesRouter {
	return &SitesRouter{
		ContextFactory: contextFactory,
		Sites:          make(qor.SitesReader),
		DomainsMap:     make(map[string]qor.SiteInterface),
		Middlewares:    route.NewMiddlewaresStack(PREFIX + ".Middlewares", true),
	}
}
func (r *SitesRouter) SetDefaultPrefix(prefix string) {
	r.DefaultPrefix = "/" + strings.Trim(prefix, "/")
}

// Use reigster a middleware to the router
func (r *SitesRouter) Use(middlewares ...*route.Middleware) {
	r.Middlewares.Add(middlewares, route.DUPLICATION_ABORT)
}

// GetMiddleware get registered middleware
func (r *SitesRouter) GetMiddleware(name string) *route.Middleware {
	return r.Middlewares.ByName[name]
}

func (sites *SitesRouter) Get(name string) qor.SiteInterface {
	return sites.Sites[name]
}

func (sites *SitesRouter) GetByDomain(host string) (site qor.SiteInterface) {
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

func (sites *SitesRouter) Each(cb func(qor.SiteInterface) bool) bool {
	for _, site := range sites.Sites {
		if !cb(site) {
			return false
		}
	}
	return true
}

func (sites *SitesRouter) EachSite(cb func(site qor.SiteInterface) error) error {
	sites.Each(func(site qor.SiteInterface) bool {
		err := cb(site)
		if err != nil {
			panic(err)
			return false
		}
		return true
	})
	return nil
}

func (sites *SitesRouter) SetupDB(setup func(db *qor.DB) error) (err error) {
	sites.Each(func(site qor.SiteInterface) bool {
		err = site.SetupDB(setup)
		if err != nil {
			return false
		}
		return true
	})
	return
}

func (sites *SitesRouter) SetupSystemDB(setup func(db *qor.DB) error) (err error) {
	sites.EachSystemDBs(func(db *qor.DB) bool {
		err = setup(db)
		if err != nil {
			return false
		}
		return true
	})
	return
}

func SiteStorageName(siteName, storageName string) string {
	return siteName + ":" + siteName
}

func (sites *SitesRouter) Register(site qor.SiteInterface) {
	sites.Sites[site.Name()] = site
	for _, domain := range site.Config().Domains {
		sites.DomainsMap[domain] = site
	}
}

func (sites *SitesRouter) EachSystemDBs(f func(db *qor.DB) bool) bool {
	return sites.Each(func(site qor.SiteInterface) bool {
		if !f(site.GetSystemDB()) {
			return false
		}
		return true
	})
}

func (sites *SitesRouter) EachDBByName(dbname string, f func(db *qor.DB) bool) bool {
	return sites.Each(func(site qor.SiteInterface) bool {
		if db := site.GetDB(dbname); db != nil {
			if !f(db) {
				return false
			}
		}
		return true
	})
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
	Handler      func(sites []qor.SiteInterface, w http.ResponseWriter, r *http.Request)
}

func (si *SitesIndex) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if si.excludes == nil {
		si.excludes = make(map[string]bool)
		for _, name := range si.ExcludeSites {
			si.excludes[name] = true
		}
		if si.URI == "" {
			si.URI = si.Router.DefaultPrefix
		}
	}

	var sites []qor.SiteInterface
	for _, site := range si.Router.Sites.Sorted() {
		if _, ok := si.excludes[site.Name()]; !ok {
			sites = append(sites, site)
		}
	}

	if si.Handler != nil {
		si.Handler(sites, w, r)
		return
	}

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
		msg += fmt.Sprintf(`<li><a href="%v/%v">%v</a></li>`, si.URI, site.Name(), site.Name())
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
