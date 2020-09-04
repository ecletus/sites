package sites

import (
	"context"
	"net/http"
	"path/filepath"
	"plugin"

	http_render "github.com/moisespsena-go/http-render"
	"github.com/moisespsena-go/http-render/ropt"

	"github.com/ecletus/plug"
	errwrap "github.com/moisespsena-go/error-wrap"
	"github.com/moisespsena-go/logging"
	"github.com/moisespsena-go/xroute"

	"github.com/ecletus/db"

	"github.com/ecletus/core"

	"github.com/moisespsena-go/aorm"
)

var E_INIT_SITE = PKG + ".init.site"

type Plugin struct {
	db.DBNames
	plug.EventDispatcher
	ContextFactoryKey,
	SitesRouterKey,
	SitesConfigKey,
	ConfigDirKey,
	SitesRegisterKey string

	sitesRouter *SitesRouter
	register    *core.SitesRegister

	config *Config
	Alone  bool
}

func (p *Plugin) RequireOptions() []string {
	return []string{p.ContextFactoryKey, p.ConfigDirKey, p.SitesConfigKey}
}

func (p *Plugin) ProvideOptions() []string {
	return []string{p.SitesRegisterKey, p.SitesRouterKey}
}

func (p *Plugin) ProvidesOptions(options *plug.Options) {
	contextFactory := options.GetInterface(p.ContextFactoryKey).(*core.ContextFactory)
	p.config = options.GetInterface(p.SitesConfigKey).(*Config)

	p.register = &core.SitesRegister{Alone: p.Alone || p.config.Alone}
	options.Set(p.SitesRegisterKey, p.register)

	p.sitesRouter = NewSitesRouter(p.register, contextFactory)
	p.sitesRouter.Prefix = p.config.Prefix
	p.sitesRouter.RedirectSiteNotFoundToIndex = p.config.RedirectSiteNotFoundToIndex
	options.Set(p.SitesRouterKey, p.sitesRouter)
}

func (p *Plugin) Init(options *plug.Options) {
	p.sitesRouter.Init()
}

func (p *Plugin) OnRegister() {
	p.On(plug.E_POST_INIT, func(e plug.PluginEventInterface) (err error) {
		sites := e.Options().GetInterface(p.SitesRouterKey).(*SitesRouter)
		func() {
			if cfg := p.config.IndexHandlerPlugin; cfg != nil {
				plug, err := plugin.Open(cfg.Path)
				if err != nil {
					log.Errorf("load index handler plugin %q failed: %s", cfg.Path, err.Error())
					return
				}
				log := logging.WithPrefix(log, "index handler ["+filepath.Base(cfg.Path)+"]")
				New, err := plug.Lookup("New")

				if err != nil {
					log.Errorf("lookup to func New() failed", err.Error())
				}

				var handler http.Handler
				switch t := New.(type) {
				case func() http.Handler:
					handler = t()
				case func(map[string]interface{}) http.Handler:
					opts := cfg.Config
					opts.Set("@log", log).
						Set("@path", cfg.Path).
						Set("@opts", e.Options())
					handler = t(opts)
				}
				if handler == nil {
					return
				}
				sites.HandleIndex = xroute.HttpHandler(handler)
			} else if p.config.IndexDir != "" {
				sites.HandleIndex = xroute.HttpHandler(http_render.New(
					ropt.Dir(p.config.IndexDir),
					ropt.DirectoryIndexEnabled(),
				))
			}
		}()

		dis := e.PluginDispatcher()
		sites.Register.OnAdd(func(site *core.Site) {
			if err = site.Init(); err == nil {
				err = dis.TriggerPlugins(&SiteEvent{plug.NewPluginEvent(ESite(site.Name())), site, e})
			}
			if err != nil {
				log.Error(err)
			}
		})
		return nil
	})

	p.On(db.E_INIT_DB, p.doDB(db.EInit))
	p.On(db.E_INIT_GORM, p.doGorm(db.EInitGorm))
	p.On(db.E_MIGRATE_DB, p.doDB(db.EMigrate))
}

func (p *Plugin) makeEventDB(ename string, site *core.Site, DB *core.DB) plug.EventInterface {
	return &db.DBEvent{plug.NewPluginEvent(ename, site), DB}
}

func (p *Plugin) makeEventGorm(ename string, site *core.Site, DB *core.DB) plug.EventInterface {
	return &db.DBEvent{plug.NewPluginEvent(ename, site), DB}
}

func (p *Plugin) do(ename func(name string) string,
	makeEvent func(ename string, site *core.Site, DB *core.DB) plug.EventInterface) func(e plug.PluginEventInterface) (err error) {
	do := func(dis plug.PluginEventDispatcherInterface, site *core.Site, DB *core.DB) (err error) {
		return dis.TriggerPlugins(makeEvent(ename(DB.Name), site, DB))
	}
	return func(e plug.PluginEventInterface) (err error) {
		if e.Name() == db.E_MIGRATE_DB {
			old := do
			do = func(dis plug.PluginEventDispatcherInterface, site *core.Site, DB *core.DB) (err error) {
				var migrator *aorm.Migrator

				if data := e.Data(); data != nil {
					if ctx, ok := e.Data().(context.Context); ok {
						if val := ctx.Value(db.OptCommitDisabled); val != nil && val.(bool) {
							DB.DB = DB.DB.Unscoped().Begin()
							migrator = aorm.NewMigrator(DB.DB)

							aorm.DefaultLogger.All(func(action string, scope *aorm.Scope) {
								log.Debug(scope.Query)
							})

							defer func() {
								DB.DB.Rollback()
							}()
						}
					}
				}
				if migrator == nil {
					migrator = DB.DB.Migrator()
					DB.DB = migrator.Db()
				}

				return migrator.Migrate(func() error {
					return old(dis, site, DB)
				})
			}
		}
		sites := e.Options().GetInterface(p.SitesRouterKey).(*SitesRouter)
		dis := e.PluginDispatcher()
		dbNames := p.GetNames()
		if len(dbNames) == 0 {
			err = sites.Register.ByName.Each(func(site *core.Site) error {
				return site.EachDB(func(DB *core.DB) error {
					return do(dis, site, DB)
				})
			})
		} else {
			err = sites.Register.ByName.Each(func(site *core.Site) (err error) {
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

type SiteEvent struct {
	plug.PluginEventInterface
	Site        *core.Site
	PluginEvent plug.PluginEventInterface
}

func ESite(name string) string {
	if name == "" {
		panic("name is blank")
	}
	return E_INIT_SITE + ":" + name
}
