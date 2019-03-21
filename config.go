package sites

import (
	"github.com/ecletus/auth/providers/facebook"
	"github.com/ecletus/auth/providers/github"
	"github.com/ecletus/auth/providers/google"
	"github.com/ecletus/auth/providers/twitter"
	"github.com/ecletus/core"
	qorconfig "github.com/ecletus/core/config"
	"github.com/moisespsena-go/stringvar"
	"github.com/moisespsena/go-error-wrap"
)

type SocialAuthConfig struct {
	Github   *github.Config
	Google   *google.Config
	Facebook *facebook.Config
	Twitter  *twitter.Config
}

func (s *SocialAuthConfig) Prepare(siteName string, args *stringvar.StringVar) {
	if s.Github != nil {
		s.Github.Name = siteName + "/" + s.Github.GetDefaultName()
	}
	if s.Google != nil {
		s.Google.Name = siteName + "/" + s.Google.GetDefaultName()
	}
	if s.Facebook != nil {
		s.Facebook.Name = siteName + "/" + s.Facebook.GetDefaultName()
	}
	if s.Twitter != nil {
		s.Twitter.Name = siteName + "/" + s.Twitter.GetDefaultName()
	}
}

type AuthConfig struct {
	UserRegistration  bool
	SocialAuthEnabled bool
	SocialAuth        *SocialAuthConfig
}

func (s *AuthConfig) Prepare(siteName string, args *stringvar.StringVar) {
	if s.SocialAuth != nil {
		s.SocialAuth.Prepare(siteName, args)
	}
}

type SiteConfig struct {
	// basic
	Name         string
	Title        string
	Domains      []string
	Db           map[string]*qorconfig.DBConfig
	MediaStorage map[string]map[string]interface{}
	RootDir      string
	SMTP         *qorconfig.SMTPConfig
	PublicURL    string
	OtherConfig  map[string]interface{}

	// advanced
	UserRegistration  bool
	SocialAuthEnabled bool
	SocialAuth        *SocialAuthConfig

	*qorconfig.SiteConfig
	*AuthConfig
}

func (s *SiteConfig) Prepare(mainConfig *Config, siteName string, args *stringvar.StringVar) error {
	if s.SiteConfig == nil {
		s.SiteConfig = &qorconfig.SiteConfig{siteName, s.Title, s.Domains, s.Db,
			s.MediaStorage, s.RootDir, s.SMTP, s.OtherConfig, s.PublicURL}
	}

	if s.AuthConfig == nil {
		s.AuthConfig = &AuthConfig{s.UserRegistration, s.SocialAuthEnabled,
			s.SocialAuth}
	}

	if s.SiteConfig.OtherConfig == nil {
		s.SiteConfig.OtherConfig = make(qorconfig.OtherConfig)
	}

	for dbName, db := range s.SiteConfig.Db {
		if mdb, ok := mainConfig.Db[dbName]; ok {
			if db.Adapter == "" {
				db.Adapter = mdb.Adapter
			}
			if db.Host == "" {
				db.Host = mdb.Host
			}
			if db.User == "" {
				db.User = mdb.User
			}
			if db.Password == "" {
				db.Password = mdb.Password
			}
			if db.Port == 0 {
				db.Port = mdb.Port
			}
			if db.SSL == "" {
				db.SSL = mdb.SSL
			}
		}
	}

	if err := s.SiteConfig.Prepare(siteName, args); err != nil {
		return errwrap.Wrap(err, "SiteConfig.Prepare")
	}
	s.AuthConfig.Prepare(siteName, args)

	oc := s.SiteConfig.OtherConfig
	oc.Set("qor:sites.siteConfig", s)
	if s.AuthConfig != nil {
		oc.SetMany("qor:auth", map[string]interface{}{
			"userRegistration": s.AuthConfig.UserRegistration,
			"social": map[string]interface{}{
				"enabled": s.AuthConfig.SocialAuth != nil && s.AuthConfig.SocialAuthEnabled,
			},
		})

		if s.AuthConfig.SocialAuth != nil {
			oc.Merge("qor:auth.social", map[string]interface{}{
				"github":   s.AuthConfig.SocialAuth.Github,
				"facebook": s.AuthConfig.SocialAuth.Facebook,
				"twitter":  s.AuthConfig.SocialAuth.Twitter,
				"google":   s.AuthConfig.SocialAuth.Google,
			})
		}
	}
	return nil
}

func (s *SiteConfig) CreateSite(cf *core.ContextFactory) core.SiteInterface {
	return core.NewSite(cf, s.SiteConfig)
}

type Config struct {
	Db            map[string]*qorconfig.DBConfig
	Host          string `env:"HOST" default:":7000"`
	Prefix        string `env:"AGHAPE_SITES_URI_PREFIX"`
	Production    bool   `env:"PRODUCTION" default:"false"`
	DefaultSite   string
	Sites         map[string]*SiteConfig
	SiteByDomain  bool `default:"false"`
	DefaultDomain string
	Alone         bool
}

func (c *Config) SystemDBAdapter() string {
	return c.Db["system"].Adapter
}
