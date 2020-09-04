package sites_loader

import (
	"fmt"

	"github.com/ecletus/plug"
	"github.com/go-errors/errors"
	errwrap "github.com/moisespsena-go/error-wrap"
	"github.com/moisespsena-go/getters"
	"github.com/moisespsena-go/maps"
	"github.com/moisespsena-go/stringvar"

	"github.com/ecletus/core"
	"github.com/ecletus/core/site_config"
	"github.com/ecletus/sites"
)

type Plugin struct {
	plug.EventDispatcher

	SitesRegisterKey,
	ContextFactoryKey,
	ConfigGettersKey,
	SitesConfigKey string

	DBNames []string
}

func (p *Plugin) RequireOptions() []string {
	return []string{p.ContextFactoryKey, p.ConfigGettersKey, p.SitesRegisterKey, p.SitesConfigKey}
}

func (p *Plugin) Init(options *plug.Options) (err error) {
	mainConfig := options.GetInterface(p.SitesConfigKey).(*sites.Config)
	defaultSiteConfig := mainConfig.SiteTemplate

	args := stringvar.New(
		"HOME", "work/home",
		"ROOT", ".",
		"DATA_DIR", mainConfig.DataDir,
		"SHARED_DATA_DIR", mainConfig.SharedDataDir(),
		"SHARED_SITE_DATA_DIR", mainConfig.SharedSiteDataDir(),
	)

	if mainConfig.Sites == nil || len(mainConfig.Sites) == 0 {
		return nil
	}
	cf := options.GetInterface(p.ContextFactoryKey).(*core.ContextFactory)
	register := options.GetInterface(p.SitesRegisterKey).(*core.SitesRegister)
	configGetter := options.GetInterface(p.ConfigGettersKey).(getters.Getter)

	for siteName, cfgi := range mainConfig.Sites {
		var cfg = make(maps.MapSI)
		if err = defaultSiteConfig.Raw.DeepCopy(cfg); err != nil {
			return errors.WrapPrefix(err, fmt.Sprintf("site %q: copy main config failed", siteName), 1)
		}
		if err = cfgi.(maps.MapSI).DeepCopy(cfg); err != nil {
			return errors.WrapPrefix(err, fmt.Sprintf("site %q: copy site config failed", siteName), 1)
		}
		delete(cfg, "sites")
		var siteConfig = &site_config.Config{Raw: cfg}
		if err = cfg.CopyTo(siteConfig); err != nil {
			return errors.WrapPrefix(err, fmt.Sprintf("site %q: unmarshall config failed", siteName), 1)
		}
		Args := args.Child("SITE_NAME", siteName)
		if err := siteConfig.Prepare(defaultSiteConfig.Db, siteName, Args); err != nil {
			return errwrap.Wrap(err, "Site %q", siteName)
		}
		site := core.NewSite(siteName, *siteConfig, configGetter, cf)
		if err := register.Add(site); err != nil {
			panic(err)
		}
	}
	return nil
}
