/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"csjhuang.net/nixos-sops-confgen/internal/config"
	"csjhuang.net/nixos-sops-confgen/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type hostColumn struct {
	Header string
	Value  func(name string, h config.Host) string
}

var defaultHostColumns = []hostColumn{
	{
		Header: "NAME",
		Value: func(name string, h config.Host) string {
			return name
		},
	},
	{
		Header: "KEY",
		Value: func(name string, h config.Host) string {
			return h.Key
		},
	},
	{
		Header: "GROUPS",
		Value: func(name string, h config.Host) string {
			if len(h.Groups) == 0 {
				return "-"
			}
			return strings.Join(h.Groups, ", ")
		},
	},
}

// getSortedKeys extracts and sorts keys from any comparable map.
func getSortedKeys[K cmp.Ordered, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func renderHosts(hosts map[string]config.Host, cols []hostColumn) {
	headers := make([]string, len(cols))
	for i, c := range cols {
		headers[i] = c.Header
	}

	names := getSortedKeys(hosts)
	rows := make([][]string, 0, len(names))

	for _, name := range names {
		row := make([]string, len(cols))
		for i, c := range cols {
			row[i] = c.Value(name, hosts[name])
		}
		rows = append(rows, row)
	}

	utils.RenderTable(headers, rows)
}

// Flags for hostAddCmd
var (
	flagHostKey    string
	flagHostGroups []string
)

var (
	hostCmd = &cobra.Command{
		Use:   "host",
		Short: "Host related operations",
	}

	hostListCmd = &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all configured hosts",
		Run: func(cmd *cobra.Command, args []string) {
			if len(Cfg.Hosts) == 0 {
				fmt.Println("No hosts configured.")
				return
			}
			renderHosts(Cfg.Hosts, defaultHostColumns)
		},
	}

	hostAddCmd = &cobra.Command{
		Use:   "add [hostname]",
		Short: "Add a new host or attach groups to an existing host",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			hostName := strings.TrimSpace(args[0])
			if hostName == "" {
				return fmt.Errorf("hostname cannot be empty")
			}

			// 0. Validate that attached groups exist in config
			for _, g := range flagHostGroups {
				if _, exists := Cfg.Groups[g]; !exists {
					return fmt.Errorf("group %q does not exist in groups section", g)
				}
			}

			// 1. Check if host already exists
			existingHost, exists := Cfg.Hosts[hostName]
			var targetHost config.Host

			if exists {
				// 2. Existing host: keep or update key, append new groups deduplicated
				targetHost = existingHost

				if flagHostKey != "" && flagHostKey != existingHost.Key {
					targetHost.Key = flagHostKey
				}

				for _, g := range flagHostGroups {
					if !slices.Contains(targetHost.Groups, g) {
						targetHost.Groups = append(targetHost.Groups, g)
					}
				}
				fmt.Printf("Updated host %q with groups: %v\n", hostName, targetHost.Groups)
			} else {
				// New host: key is mandatory
				if flagHostKey == "" {
					return fmt.Errorf("flag --key is required when registering a new host")
				}
				targetHost = config.Host{
					Key:    flagHostKey,
					Groups: flagHostGroups,
				}
				fmt.Printf("Created new host %q\n", hostName)
			}

			// Update in-memory state
			if Cfg.Hosts == nil {
				Cfg.Hosts = make(map[string]config.Host)
			}
			Cfg.Hosts[hostName] = targetHost

			// 3. Write only to the specific Viper node path
			viper.Set(fmt.Sprintf("hosts.%s", hostName), map[string]any{
				"key":    targetHost.Key,
				"groups": targetHost.Groups,
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
	rootCmd.AddCommand(hostCmd)
	hostCmd.AddCommand(hostListCmd)
	hostCmd.AddCommand(hostAddCmd)

	hostAddCmd.Flags().StringVarP(&flagHostKey, "key", "k", "", "age public key for the host")
	hostAddCmd.Flags().StringSliceVarP(&flagHostGroups, "groups", "g", []string{}, "comma-separated groups or multiple -g flags")
}
