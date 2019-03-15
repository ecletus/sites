package sites

import (
	"fmt"
	"strings"

	"github.com/aghape/core"
	"github.com/moisespsena/go-error-wrap"
	"github.com/spf13/cobra"
)

type CmdUtils struct {
	SitesReader core.SitesReaderInterface
}

func (cu *CmdUtils) Site(command *cobra.Command, run ...func(cmd *cobra.Command, site core.SiteInterface, args []string) error) *cobra.Command {
	Args := command.Args
	command.Args = func(cmd *cobra.Command, args []string) (err error) {
		err = cobra.MinimumNArgs(1)(cmd, args)
		if err == nil && cu.SitesReader.Get(args[0]) == nil {
			return fmt.Errorf("Site %q does not exists.\n", args[0])
		}

		if Args != nil {
			return Args(cmd, args[1:])
		}
		return
	}
	if len(run) == 1 {
		command.RunE = func(cmd *cobra.Command, args []string) error {
			return run[0](cmd, cu.SitesReader.Get(args[0]), args[1:])
		}
	}

	UseParts := strings.Split(command.Use, " ")
	command.Use = strings.Join(append([]string{UseParts[0], "SITE_NAME"}, UseParts[1:]...), " ")
	return command
}

func (cu *CmdUtils) Sites(command *cobra.Command, run ...func(cmd *cobra.Command, site core.SiteInterface, args []string) error) *cobra.Command {
	if len(run) == 1 {
		command.RunE = func(cmd *cobra.Command, args []string) error {
			siteName, err := cmd.Flags().GetString("site-name")
			if err != nil {
				return err
			}
			
			callSite := func(site core.SiteInterface) error {
				defer func() {
					site.EachDB(func(db *core.DB) error {
						db.Raw.Close()
						return nil
					})
				}()
				err := run[0](cmd, site, args)
				if err != nil {
					return errwrap.Wrap(err, "Site %q", site.Name())
				}
				return nil
			}

			if siteName == "*" {
				return cu.SitesReader.Each(callSite)
			} else {
				site := cu.SitesReader.Get(siteName)
				return callSite(site)
			}
		}
	}
	command.Flags().String("site-name", "*", "the site name. Use * (asterisk) for all sites")
	return command
}
