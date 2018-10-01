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
	Args := command.Args
	command.Args = func(cmd *cobra.Command, args []string) (err error) {
		if len(args) == 0 {
			if siteNames := cu.SitesReader.Names(); len(siteNames) == 1 {
				args = append(args, siteNames[0])
			} else {
				return
			}
		}
		if args[0] != "*" && cu.SitesReader.Get(args[0]) == nil {
			return fmt.Errorf("Site %q does not exists.\n", args[0])
		}

		if Args != nil {
			return Args(cmd, args[1:])
		}
		return
	}
	if len(run) == 1 {
		command.RunE = func(cmd *cobra.Command, args []string) error {
			callSite := func(site core.SiteInterface) error {
				defer func() {
					site.EachDB(func(db *core.DB) bool {
						db.Raw.Close()
						return true
					})
				}()
				err := run[0](cmd, site, args)
				if err != nil {
					return errwrap.Wrap(err, "Site %q", site.Name())
				}
				return nil
			}

			if len(args) == 0 {
				return cu.SitesReader.Each(func(site core.SiteInterface) (cont bool, err error) {
					err = callSite(site)
					return err == nil, err
				})
			} else {
				site := cu.SitesReader.Get(args[0])
				args = args[1:]
				return callSite(site)
			}
		}
	}

	UseParts := strings.Split(command.Use, " ")
	command.Use = strings.Join(append([]string{UseParts[0], "[SITE_NAME]"}, UseParts[1:]...), " ")
	return command
}
