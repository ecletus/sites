package sites

import (
	"github.com/aghape/core"
	"github.com/aghape/db"
	"github.com/aghape/plug"
	"github.com/moisespsena/go-error-wrap"
)

var E_INIT_SITE = PKG + ".init.site"

type Plugin struct {
	db.DBNames
	plug.EventDispatcher
	ContextFactoryKey, SitesRouterKey, SitesReaderKey string
}

func (p *Plugin) RequireOptions() []string {
	return []string{p.ContextFactoryKey}
}

func (p *Plugin) ProvideOptions() []string {
	return []string{p.SitesRouterKey, p.SitesReaderKey}
}

type SiteEvent struct {
	plug.PluginEventInterface
	Site        core.SiteInterface
	PluginEvent plug.PluginEventInterface
}

func ESite(name string) string {
	if name == "" {
		panic("name is blank")
	}
	return E_INIT_SITE + ":" + name
}

func (p *Plugin) Init(options *plug.Options) {
	cf := options.GetInterface(p.ContextFactoryKey).(*core.ContextFactory)
	sites := NewSites(cf)
	options.Set(p.SitesRouterKey, sites)
	options.Set(p.SitesReaderKey, sites.Sites)
}

func (p *Plugin) makeEventDB(ename string, site core.SiteInterface, DB *core.DB) plug.EventInterface {
	return &db.DBEvent{plug.NewPluginEvent(ename, site), DB}
}

func (p *Plugin) makeEventGorm(ename string, site core.SiteInterface, DB *core.DB) plug.EventInterface {
	return &db.DBEvent{plug.NewPluginEvent(ename, site), DB}
}

func (p *Plugin) do(ename func(name string) string,
	makeEvent func(ename string, site core.SiteInterface, DB *core.DB) plug.EventInterface) func(e plug.PluginEventInterface) (err error) {
	do := func(dis plug.PluginEventDispatcherInterface, site core.SiteInterface, DB *core.DB) (err error) {
		return dis.TriggerPlugins(makeEvent(ename(DB.Name), site, DB))
	}
	return func(e plug.PluginEventInterface) (err error) {
		sites := e.Options().GetInterface(p.SitesRouterKey).(*SitesRouter)
		dis := e.PluginDispatcher()
		dbNames := p.GetNames()
		if len(dbNames) == 0 {
			err = sites.Sites.Each(func(site core.SiteInterface) error {
				return site.EachDB(func(DB *core.DB) error {
					return do(dis, site, DB)
				})
			})
		} else {
			err = sites.Sites.Each(func(site core.SiteInterface) (err error) {
				for _, dbName := range dbNames {
					if DB := site.GetDB(dbName); DB != nil {
						if err = do(dis, site, DB); err != nil {
							return errwrap.Wrap(err, dbName)
						}
					}
				}
				return nil
			})
		}
		return
	}
}

func (p *Plugin) doDB(ename func(name string) string) func(e plug.PluginEventInterface) (err error) {
	return p.do(ename, p.makeEventDB)
}

func (p *Plugin) doGorm(ename func(name string) string) func(e plug.PluginEventInterface) (err error) {
	return p.do(ename, p.makeEventGorm)
}

func (p *Plugin) OnRegister() {
	p.On(plug.E_POST_INIT, func(e plug.PluginEventInterface) (err error) {
		sites := e.Options().GetInterface(p.SitesRouterKey).(*SitesRouter)
		dis := e.PluginDispatcher()
		return sites.Each(func(site core.SiteInterface) error {
			return dis.TriggerPlugins(&SiteEvent{plug.NewPluginEvent(ESite(site.Name())), site, e})
		})
	})

	p.On(db.E_INIT_DB, p.doDB(db.EInit))
	p.On(db.E_INIT_GORM, p.doGorm(db.EInitGorm))
	p.On(db.E_MIGRATE_DB, p.doDB(db.EMigrate))
}
