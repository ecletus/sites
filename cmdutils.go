package sites

import (
	"fmt"
	"strings"

	errwrap "github.com/moisespsena-go/error-wrap"
	"github.com/spf13/cobra"

	"github.com/ecletus/core"
)

type CmdUtils struct {
	SitesRegister *core.SitesRegister
}

func (cu *CmdUtils) Site(command *cobra.Command, run ...func(cmd *cobra.Command, site *core.Site, args []string) error) *cobra.Command {
	Args := command.Args
	command.Args = func(cmd *cobra.Command, args []string) (err error) {
		err = cobra.MinimumNArgs(1)(cmd, args)
		if err == nil && cu.SitesRegister.MustGet(args[0]) == nil {
			return fmt.Errorf("Site %q does not exists.\n", args[0])
		}

		if Args != nil {
			return Args(cmd, args[1:])
		}
		return
	}
	if len(run) == 1 {
		command.RunE = func(cmd *cobra.Command, args []string) error {
			return run[0](cmd, cu.SitesRegister.MustGet(args[0]), args[1:])
		}
	}

	UseParts := strings.Split(command.Use, " ")
	command.Use = strings.Join(append([]string{UseParts[0], "SITE_NAME"}, UseParts[1:]...), " ")
	return command
}

func (cu *CmdUtils) Sites(command *cobra.Command, run ...func(cmd *cobra.Command, site *core.Site, args []string) error) *cobra.Command {
	if len(run) == 1 {
		var oldArgs = command.Args
		use := strings.Split(command.Use, " ")
		command.Use = strings.Join(append([]string{use[0], "SITE_NAME[,SITE_NAME...]"}, use[1:]...), " ")
		command.Args = func(cmd *cobra.Command, args []string) (err error) {
			if err = cobra.MinimumNArgs(1)(cmd, args); err == nil {
				if args[0] != "*" {
					var siteNames []string
					for _, siteName := range strings.Split(args[0], ",") {
						if siteName = strings.TrimSpace(siteName); siteName == "" || siteName == "*" {
							continue
						}
						if !cu.SitesRegister.Has(siteName) {
							return fmt.Errorf("Site %q does not exists.\n", siteName)
						}
						siteNames = append(siteNames, siteName)
					}
					args[0] = strings.Join(siteNames, ",")
				}
				if oldArgs != nil {
					return oldArgs(cmd, args[1:])
				}
			}
			return
		}
		command.RunE = func(cmd *cobra.Command, args []string) (err error) {
			callSite := func(site *core.Site) error {
				err := run[0](cmd, site, args)
				if err != nil {
					return errwrap.Wrap(err, "Site %q", site.Name())
				}
				return nil
			}
			siteNames := strings.Split(args[0], ",")
			args = args[1:]

			if siteNames[0] == "*" {
				siteNames = cu.SitesRegister.ByName.Names()
			}
			for _, siteName := range siteNames {
				if err = cu.SitesRegister.Only(siteName, callSite); err != nil {
					return
				}
			}
			return nil
		}
	}
	return command
}

func (cu *CmdUtils) Alone(command *cobra.Command, run ...func(cmd *cobra.Command, site *core.Site, args []string) error) *cobra.Command {
	if len(run) == 1 {
		command.Args = func(cmd *cobra.Command, args []string) (err error) {
			if !cu.SitesRegister.Alone {
				return fmt.Errorf("require sites alone mode.")
			}
			if !cu.SitesRegister.HasSites() {
				return fmt.Errorf("no site registered.")
			}
			return
		}
		command.RunE = func(cmd *cobra.Command, args []string) (err error) {
			site := cu.SitesRegister.Site()

			if err = run[0](cmd, site, args); err != nil {
				return errwrap.Wrap(err, "Site %q", site.Name())
			}
			return nil
		}
	}
	return command
}
