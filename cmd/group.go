/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"strings"

	"csjhuang.net/nixos-sops-confgen/internal/config"
	"csjhuang.net/nixos-sops-confgen/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type groupColumn struct {
	Header string
	Value  func(name string, g config.Group) string
}

var defaultGroupColumns = []groupColumn{
	{
		Header: "NAME",
		Value: func(name string, g config.Group) string {
			return name
		},
	},
	{
		Header: "PATH_REGEX",
		Value: func(name string, g config.Group) string {
			if g.PathRegex == "" {
				return "-"
			}
			return g.PathRegex
		},
	},
}

func renderGroups(groups map[string]config.Group, cols []groupColumn) {
	headers := make([]string, len(cols))
	for i, c := range cols {
		headers[i] = c.Header
	}

	names := getSortedKeys(groups)
	rows := make([][]string, 0, len(names))

	for _, name := range names {
		row := make([]string, len(cols))
		for i, c := range cols {
			row[i] = c.Value(name, groups[name])
		}
		rows = append(rows, row)
	}

	utils.RenderTable(headers, rows)
}

// Flags for groupAddCmd
var flagGroupPathRegex string

var (
	groupCmd = &cobra.Command{
		Use:   "group",
		Short: "Group related operations",
	}

	groupListCmd = &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all configured groups",
		Run: func(cmd *cobra.Command, args []string) {
			if len(Cfg.Groups) == 0 {
				fmt.Println("No groups configured.")
				return
			}
			renderGroups(Cfg.Groups, defaultGroupColumns)
		},
	}

	groupAddCmd = &cobra.Command{
		Use:   "add [group]",
		Short: "Add a new group or update an existing group's path_regex",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			groupName := strings.TrimSpace(args[0])
			if groupName == "" {
				return fmt.Errorf("group name cannot be empty")
			}

			existing, exists := Cfg.Groups[groupName]
			pathRegex := strings.TrimSpace(flagGroupPathRegex)

			var target config.Group
			switch {
			case !exists:
				if pathRegex == "" {
					return fmt.Errorf("flag --path-regex is required when creating a new group")
				}
				target = config.Group{PathRegex: pathRegex}
				fmt.Printf("Created new group %q\n", groupName)
			case pathRegex != "" && pathRegex != existing.PathRegex:
				target = existing
				target.PathRegex = pathRegex
				fmt.Printf("Updated group %q\n", groupName)
			default:
				fmt.Printf("Group %q already up to date\n", groupName)
				return nil
			}

			// Update in-memory state
			if Cfg.Groups == nil {
				Cfg.Groups = make(map[string]config.Group)
			}
			Cfg.Groups[groupName] = target

			// Write only to the specific Viper node path
			viper.Set(fmt.Sprintf("groups.%s", groupName), map[string]any{
				"path_regex": target.PathRegex,
			})

			if err := viper.WriteConfig(); err != nil {
				return fmt.Errorf("failed to save configuration: %w", err)
			}

			fmt.Println("Configuration successfully updated.")
			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(groupCmd)
	groupCmd.AddCommand(groupListCmd)
	groupCmd.AddCommand(groupAddCmd)

	groupAddCmd.Flags().StringVarP(&flagGroupPathRegex, "path-regex", "p", "", "path regex the group applies to")
}
