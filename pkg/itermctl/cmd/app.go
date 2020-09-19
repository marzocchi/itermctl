package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	"mrz.io/itermctl/pkg/cli"
	"mrz.io/itermctl/pkg/itermctl"
	"mrz.io/itermctl/pkg/itermctl/env"
	"os"
	"text/tabwriter"
)

var AppCommand = &cli.NestedCommand{
	Command: &cobra.Command{
		Use:   "app",
		Short: "Commands for interacting with the running iTerm2 application",
	},
	Subcommands: []*cli.NestedCommand{
		SendTextCommand,
		ListSessionsCommand,
		SplitPanesCommand,
		CreateTabCommand,
	},
}

var SendTextCommand = &cli.NestedCommand{
	Command: &cobra.Command{
		Use:  "send-text SESSION_ID",
		Args: cobra.ExactArgs(1),
		RunE: cli.WithApp("itermctl", func(app *itermctl.App, cmd *cobra.Command, args []string) error {
			suppressBroadcast, err := cmd.Flags().GetBool("suppress-broadcast")
			if err != nil {
				return err
			}

			data, err := ioutil.ReadAll(os.Stdin)
			if err != nil {
				return err
			}

			session := app.Session(args[0])
			if session == nil {
				panic(fmt.Sprintf("no session: %s", args[0]))
			}

			if err := session.SendText(string(data), suppressBroadcast); err != nil {
				return err
			}

			return nil
		}),
	},
}

var ListSessionsCommand = &cli.NestedCommand{
	Command: &cobra.Command{
		Use:  "list-sessions",
		Long: "list sessions with status ([i]nactive, [a]ctive, [b]buried), session ID, window ID and optionally the session status",
		Args: cobra.ExactArgs(0),
		RunE: cli.WithApp("itermctl", func(app *itermctl.App, cmd *cobra.Command, args []string) error {
			withTitles, err := cmd.Flags().GetBool("titles")
			if err != nil {
				return err
			}

			sessionsResponse, err := app.ListSessions()
			if err != nil {
				return err
			}

			activeSessionId := app.ActiveSession().Id()

			tw := tabwriter.NewWriter(os.Stdout, 0, 8, 1, ' ', 0)
			for _, w := range sessionsResponse.GetWindows() {
				for _, t := range w.GetTabs() {
					for _, l := range t.Root.Links {
						var flag string
						var title string

						if withTitles {
							title = l.GetSession().GetTitle()
						}

						if l.GetSession().GetUniqueIdentifier() == activeSessionId {
							flag = "a"
						} else {
							flag = "i"
						}

						_, _ = fmt.Fprintf(tw,
							"%s\t%s\t%s\t%s\n",
							flag,
							l.GetSession().GetUniqueIdentifier(),
							w.GetWindowId(),
							title,
						)
					}
				}
			}

			for _, bs := range sessionsResponse.GetBuriedSessions() {
				var title string
				if withTitles {
					title = bs.GetTitle()
				}
				_, _ = fmt.Fprintf(
					tw,
					"b\t%s\t\t%s\n",
					bs.GetUniqueIdentifier(),
					title,
				)
			}

			return tw.Flush()
		}),
	},
}

var SplitPanesCommand = &cli.NestedCommand{
	Command: &cobra.Command{
		Use:  "split-pane SESSION_ID",
		Args: cobra.MaximumNArgs(1),
		RunE: cli.WithApp("itermctl", func(app *itermctl.App, cmd *cobra.Command, args []string) error {
			before, err := cmd.Flags().GetBool("before")
			if err != nil {
				return err
			}

			vertical, err := cmd.Flags().GetBool("vertical")
			if err != nil {
				return err
			}

			var sessionId string

			if len(args) > 0 {
				sessionId = args[0]
			} else {
				shellSession, err := env.Session()
				if err != nil {
					return fmt.Errorf("can't find current session: %w", err)
				}
				sessionId = shellSession.SessionId
			}

			session := app.Session(sessionId)
			if session == nil {
				panic(fmt.Sprintf("no session: %s", sessionId))
			}

			sessionIds, err := session.SplitPane(vertical, before)
			if err != nil {
				return err
			}

			for _, sessionId := range sessionIds {
				fmt.Printf("%s\n", sessionId)
			}

			return nil
		}),
	},
}

var CreateTabCommand = &cli.NestedCommand{
	Command: &cobra.Command{
		Use:  "create-tab WINDOW_ID",
		Args: cobra.MaximumNArgs(1),
		RunE: cli.WithApp("itermctl", func(app *itermctl.App, cmd *cobra.Command, args []string) error {
			var windowId string
			if len(args) > 0 {
				windowId = args[0]
			} else {
				shellSession, err := env.Session()
				if err != nil {
					return fmt.Errorf("can't find current window: %w", err)
				}

				sessions, err := app.ListSessions()
				if err != nil {
					return fmt.Errorf("can't find current window: %w", err)
				}

				if shellSession.WindowIndex > len(sessions.GetWindows())-1 {
					return fmt.Errorf("can't find current window: there are only %d windows but index from shell is %d",
						len(sessions.GetWindows()), shellSession.WindowIndex)
				}

				window := sessions.GetWindows()[shellSession.WindowIndex]
				windowId = window.GetWindowId()
			}

			tabIndex, err := cmd.Flags().GetUint32("tab-index")
			if err != nil {
				return err
			}

			profileName, err := cmd.Flags().GetString("profile-name")
			if err != nil {
				return err
			}

			resp, err := app.CreateTab(windowId, tabIndex, profileName)
			if err != nil {
				return err
			}

			fmt.Printf("%s\n", resp.GetSessionId())
			return nil
		}),
	},
}

func init() {
	AppCommand.PersistentFlags().Bool("activate", false,
		"start iTerm2 and bring it to the front when requesting cookie; has no effect when the ITERM2_COOKIE environment variable is set")

	SendTextCommand.Flags().Bool("suppress-broadcast", true, "")

	SplitPanesCommand.Flags().Bool("vertical", false, "")
	SplitPanesCommand.Flags().Bool("before", false, "")

	CreateTabCommand.Flags().String("profile-name", "Default", "")
	CreateTabCommand.Flags().Uint32("tab-index", 0, "")

	ListSessionsCommand.Flags().Bool("titles", false, "also list session titles")
}
