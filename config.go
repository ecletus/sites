package sites

import (
	"github.com/aghape/auth/providers/facebook"
	"github.com/aghape/auth/providers/github"
	"github.com/aghape/auth/providers/google"
	"github.com/aghape/auth/providers/twitter"
	"github.com/aghape/aghape"
	qorconfig "github.com/aghape/aghape/config"
)

type SocialAuthConfig struct {
	Github   *github.Config
	Google   *google.Config
	Facebook *facebook.Config
	Twitter  *twitter.Config
}

func (s *SocialAuthConfig) Prepare(siteName string, args *qorconfig.Args) {
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

func (s *AuthConfig) Prepare(siteName string, args *qorconfig.Args) {
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
	MediaStorage map[string]*qorconfig.MediaStorageConfig
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

func (s *SiteConfig) Prepare(siteName string, args *qorconfig.Args) {
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

	s.SiteConfig.Prepare(siteName, args)
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
}

func (s *SiteConfig) CreateSite(cf *qor.ContextFactory) qor.SiteInterface {
	return qor.NewSite(cf, s.SiteConfig)
}

type Config struct {
	Host          string `env:"HOST" default:":7000"`
	Prefix        string `env:"PREFIX"`
	Production    bool   `env:"PRODUCTION" default:false`
	DefaultSite   *SiteConfig
	Sites         map[string]*SiteConfig
	SiteByDomain  bool   `default:false`
	DefaultDomain string
}
