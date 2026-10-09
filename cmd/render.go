/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"text/template"

	"csjhuang.net/nixos-sops-confgen/internal/config"
	"csjhuang.net/nixos-sops-confgen/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

// sopsTemplate renders the .sops.yaml file. The keys section only exists to
// define YAML anchors; creation_rules reference them so recipients stay in one
// place. Hosts and rules are emitted in config declaration order.
const sopsTemplate = `---
keys:
  - users:
      - &personal_enc_pgp_key {{ .UserKey }}
  - hosts:{{ range .Hosts }}
      - &{{ .Name }} {{ .Key }}{{ end }}
creation_rules:{{ range .Rules }}
  - path_regex: {{ .PathRegex }}
    key_groups:
      - pgp:
          - *personal_enc_pgp_key{{ if .Hosts }}
        age:{{ range .Hosts }}
          - *{{ .Name }}{{ end }}{{ end }}{{ end }}
`

var sopsTmpl = template.Must(template.New("sops").Parse(sopsTemplate))

// templateHost is a host anchor in the rendered document.
type templateHost struct {
	Name string
	Key  string
}

// templateRule is one creation rule and the hosts allowed to decrypt it.
type templateRule struct {
	PathRegex string
	Hosts     []templateHost
}

// templateData is the view model handed to sopsTmpl.
type templateData struct {
	UserKey string
	Hosts   []templateHost
	Rules   []templateRule
}

// configOrder holds the declaration order of hosts and groups as written in the
// config file, which viper's map unmarshaling discards.
type configOrder struct {
	hosts  []string
	groups []string
}

// loadConfigOrder reads path and returns the order in which hosts and groups
// were declared.
func loadConfigOrder(path string) (configOrder, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return configOrder{}, err
	}

	var doc struct {
		Groups yaml.Node `yaml:"groups"`
		Hosts  yaml.Node `yaml:"hosts"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return configOrder{}, err
	}

	return configOrder{
		hosts:  mappingKeys(doc.Hosts),
		groups: mappingKeys(doc.Groups),
	}, nil
}

// mappingKeys returns the keys of a YAML mapping node in declaration order.
func mappingKeys(node yaml.Node) []string {
	if node.Kind != yaml.MappingNode {
		return nil
	}
	keys := make([]string, 0, len(node.Content)/2)
	for i := 0; i+1 < len(node.Content); i += 2 {
		keys = append(keys, node.Content[i].Value)
	}
	return keys
}

// orderedKeys returns keys ordered as in order, followed by any remaining keys
// (sorted) so nothing is silently dropped.
func orderedKeys[V any](order []string, m map[string]V) []string {
	seen := make(map[string]bool, len(order))
	keys := make([]string, 0, len(m))
	for _, k := range order {
		if _, ok := m[k]; ok && !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}

	var rest []string
	for k := range m {
		if !seen[k] {
			rest = append(rest, k)
		}
	}
	slices.Sort(rest)
	return append(keys, rest...)
}

// buildTemplateData derives the rendered document from the config.
func buildTemplateData(cfg config.Config, order configOrder) templateData {
	hostOrder := orderedKeys(order.hosts, cfg.Hosts)
	groupOrder := orderedKeys(order.groups, cfg.Groups)

	hosts := make([]templateHost, 0, len(hostOrder))
	for _, name := range hostOrder {
		hosts = append(hosts, templateHost{Name: name, Key: cfg.Hosts[name].Key})
	}

	rules := make([]templateRule, 0, len(groupOrder))
	for _, group := range groupOrder {
		rule := templateRule{PathRegex: cfg.Groups[group].PathRegex}
		for _, host := range hosts {
			if slices.Contains(cfg.Hosts[host.Name].Groups, group) {
				rule.Hosts = append(rule.Hosts, host)
			}
		}
		rules = append(rules, rule)
	}

	return templateData{
		UserKey: cfg.User.PersonalEncGpgKey,
		Hosts:   hosts,
		Rules:   rules,
	}
}

// renderSops executes the sops template for the given config.
func renderSops(cfg config.Config, order configOrder) ([]byte, error) {
	if cfg.User.PersonalEncGpgKey == "" {
		return nil, fmt.Errorf("user.personal_enc_pgp_key is not set")
	}

	var buf bytes.Buffer
	if err := sopsTmpl.Execute(&buf, buildTemplateData(cfg, order)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Flags for renderCmd
var (
	flagRenderInplace bool
	flagRenderOutput  string
)

var renderCmd = &cobra.Command{
	Use:   "render",
	Short: "Render the .sops.yaml file from the config",
	Long: `Render the .sops.yaml creation rules from config.yaml.

By default the command prints a unified diff against the existing output file
without modifying it. Pass --inplace to write the rendered result back.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		order, err := loadConfigOrder(viper.ConfigFileUsed())
		if err != nil {
			return fmt.Errorf("failed to read config order: %w", err)
		}

		rendered, err := renderSops(Cfg, order)
		if err != nil {
			return fmt.Errorf("failed to render sops config: %w", err)
		}

		existing, err := os.ReadFile(flagRenderOutput)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to read %s: %w", flagRenderOutput, err)
		}

		if flagRenderInplace {
			if bytes.Equal(existing, rendered) {
				fmt.Printf("%s is already up to date.\n", flagRenderOutput)
				return nil
			}
			if dir := filepath.Dir(flagRenderOutput); dir != "." && dir != "" {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					return fmt.Errorf("failed to create %s: %w", dir, err)
				}
			}
			if err := os.WriteFile(flagRenderOutput, rendered, 0o644); err != nil {
				return fmt.Errorf("failed to write %s: %w", flagRenderOutput, err)
			}
			fmt.Printf("Wrote %s\n", flagRenderOutput)
			return nil
		}

		diff := utils.UnifiedDiff(string(existing), string(rendered), flagRenderOutput)
		if diff == "" {
			fmt.Println("No changes.")
			return nil
		}
		fmt.Print(diff)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(renderCmd)

	renderCmd.Flags().BoolVarP(&flagRenderInplace, "inplace", "i", false, "write the rendered config back to the output file")
	renderCmd.Flags().StringVarP(&flagRenderOutput, "output", "o", ".sops.yaml", "path of the .sops.yaml file to render")
}
