package sites

import (
	"path/filepath"

	"github.com/ecletus/core/db/dbconfig"

	"github.com/moisespsena-go/maps"
	"github.com/moisespsena-go/stringvar"
)

type SocialAuthConfig struct {
	/*Github   *github.Config
	Google   *google.Config
	Facebook *facebook.Config
	Twitter  *twitter.Config*/
}

func (s *SocialAuthConfig) Prepare(siteName string, args *stringvar.StringVar) {
	/*if s.Github != nil {
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
	*/
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

type ConfigPluginHandler struct {
	Path   string     `mapstructure:"path"`
	Config maps.MapSI `mapstructure:"config"`
}

type SiteConfig struct {
	Db  map[string]*dbconfig.DBConfig `mapstructure:"db"`
	Raw maps.MapSI
}

func (this *SiteConfig) SystemDBAdapter() string {
	return this.Db["system"].Adapter
}

type Config struct {
	DefaultSite       string `mapstructure:"default_site"`
	Alone             bool   `mapstructure:"alone"`
	Prefix            string `mapstructure:"prefix"`
	DataDir           string `mapstructure:"data_dir"`
	SingularTableName bool   `mapstructure:"singular_table_name"`
	// IndexHandlerPlugin vgo plugin for xroute.ContextHandler
	IndexHandlerPlugin          *ConfigPluginHandler `mapstructure:"index_handler_plugin"`
	IndexDir                    string               `mapstructure:"index_dir"`
	RedirectSiteNotFoundToIndex bool                 `mapstructure:"redirect_site_not_found_to_index"`
	LogPath                     string               `mapstructure:"log_path"`
	SiteTemplate                SiteConfig           `mapstructure:"site_template"`
	Raw                         maps.MapSI
	Sites                       maps.MapSI
}

func (this Config) SharedDataDir() string {
	return filepath.Join(this.DataDir, "_shared")
}

func (this Config) SharedSiteDataDir() string {
	return filepath.Join(this.DataDir, "_shared", "site")
}
