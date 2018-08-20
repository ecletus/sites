package sites

import (
	"github.com/aghape/aghape"
	"github.com/aghape/db"
	"github.com/aghape/plug"
)

var E_INIT_SITE = PREFIX + ".init.site"

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
	Site        qor.SiteInterface
	PluginEvent plug.PluginEventInterface
}

func ESite(name string) string {
	if name == "" {
		panic("name is blank")
	}
	return E_INIT_SITE + ":" + name
}

func (p *Plugin) Init(options *plug.Options) {
	cf := options.GetInterface(p.ContextFactoryKey).(*qor.ContextFactory)
	sites := NewSites(cf)
	options.Set(p.SitesRouterKey, sites)
	options.Set(p.SitesReaderKey, sites.Sites)
}

func (p *Plugin) makeEventDB(ename string, site qor.SiteInterface, DB *qor.DB) plug.EventInterface {
	return &db.DBEvent{plug.NewPluginEvent(ename, site), DB}
}

func (p *Plugin) makeEventGorm(ename string, site qor.SiteInterface, DB *qor.DB) plug.EventInterface {
	return &db.GormDBEvent{plug.NewPluginEvent(ename, site), DB.DB}
}

func (p *Plugin) do(ename func(name string) string, makeEvent func(ename string, site qor.SiteInterface, DB *qor.DB) plug.EventInterface) func(e plug.PluginEventInterface) (err error) {
	return func(e plug.PluginEventInterface) (err error) {
		sites := e.Options().GetInterface(p.SitesRouterKey).(*SitesRouter)
		dis := e.PluginDispatcher()
		dbNames := p.GetNames()
		if len(dbNames) == 0 {
			sites.Each(func(site qor.SiteInterface) bool {
				return site.EachDB(func(DB *qor.DB) bool {
					err = dis.TriggerPlugins(makeEvent(ename(DB.Name), site, DB))
					return err == nil
				})
			})
		} else {
			sites.Each(func(site qor.SiteInterface) bool {
				for _, dbName := range dbNames {
					if DB := site.GetDB(dbName); DB != nil {
						err = dis.TriggerPlugins(makeEvent(ename(DB.Name), site, DB))
						if err != nil {
							return false
						}
					}
				}
				return true
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
		sites.Each(func(site qor.SiteInterface) bool {
			err = dis.TriggerPlugins(&SiteEvent{plug.NewPluginEvent(ESite(site.Name())), site, e})
			return err == nil
		})
		return
	})
	p.On(db.E_INIT_DB, p.doDB(db.EInit))
	p.On(db.E_INIT_GORM, p.doGorm(db.EInitGorm))

	p.On(db.E_MIGRATE_DB, p.doDB(db.EMigrate))
	p.On(db.E_MIGRATE_GORM, p.doGorm(db.EMigrateGorm))
}
