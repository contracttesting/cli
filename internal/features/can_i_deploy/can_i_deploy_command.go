package can_i_deploy

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/contracttesting/cli/internal/components"
	"github.com/contracttesting/cli/internal/gitversion"
	"github.com/spf13/cobra"
)

const requestTimeout = 30 * time.Second

var ErrSilent = components.ErrSilent

func NewCanIDeployCommand(client *CanIDeployClient) *cobra.Command {
	commandHandler := func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true
		command.SilenceErrors = true

		participant := args[0]

		version, err := command.Flags().GetString("version")
		if err != nil {
			return fmt.Errorf("get version: %w", err)
		}

		if version == "" {
			if version, err = gitversion.Resolve("."); err != nil {
				return err
			}
		}

		environment, err := command.Flags().GetString("environment")
		if err != nil {
			return fmt.Errorf("get environment: %w", err)
		}

		ctx, cancel := context.WithTimeout(command.Context(), requestTimeout)
		defer cancel()

		requestBody := &CanIDeployRequestBody{
			Participant: participant,
			Version:     version,
			Environment: environment,
		}

		resp, err := client.Check(ctx, requestBody)
		if err != nil {
			fmt.Fprintf(command.ErrOrStderr(), "❌ %s\n", err.Error())
			return ErrSilent
		}

		if !resp.Deployable {
			fmt.Fprint(command.OutOrStdout(), formatNotDeployableReport(participant, environment, resp.Results))
			return ErrSilent
		}

		fmt.Fprintf(command.OutOrStdout(), "🚀 %s can be deployed to %s\n", participant, environment)

		return nil
	}

	command := &cobra.Command{
		Use:   "can-i-deploy [participant]",
		Short: "Check whether a participant version can be deployed to an environment",
		Args:  cobra.ExactArgs(1),
		RunE:  commandHandler,
	}

	command.Flags().String("version", "", "Version to check, e.g. a commit hash or semver tag (default: git rev-parse --short HEAD)")
	command.Flags().String("environment", "", "Target environment name (required)")
	command.MarkFlagRequired("environment")

	return command
}

func formatNotDeployableReport(participant, environment string, results map[string]CanIDeployResult) string {
	var report strings.Builder
	fmt.Fprintf(&report, "❌ %s cannot be deployed to %s\n", participant, environment)

	counterparts := make([]string, 0, len(results))
	for name, result := range results {
		if !result.Deployable {
			counterparts = append(counterparts, name)
		}
	}
	sort.Strings(counterparts)

	for _, name := range counterparts {
		result := results[name]

		report.WriteString("\n" + name)
		if result.ParticipantVersion != nil {
			fmt.Fprintf(&report, " (%s, latest deployed version in %q)", *result.ParticipantVersion, environment)
		}
		report.WriteString(":\n")

		endpoints := sortedKeys(result.Endpoints)
		for _, endpoint := range endpoints {
			methods := sortedKeys(result.Endpoints[endpoint])
			for _, method := range methods {
				fmt.Fprintf(&report, "  %s %s\n", strings.ToUpper(method), endpoint)

				interactions := result.Endpoints[endpoint][method]
				for _, interaction := range sortedInteractions(interactions) {
					if interaction == "request" {
						report.WriteString("    request:\n")
					} else {
						fmt.Fprintf(&report, "    response %s:\n", interaction)
					}

					lines := make([]string, 0, len(interactions[interaction]))
					for _, contractBreak := range interactions[interaction] {
						lines = append(lines, formatBreakLine(environment, interaction, contractBreak))
					}
					sort.Strings(lines)
					for _, line := range lines {
						fmt.Fprintf(&report, "      - %s\n", line)
					}
				}
			}
		}
	}

	return report.String()
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// sortedInteractions orders a method's interaction keys as "request" first, then the
// response status codes sorted.
func sortedInteractions(interactions map[string][]ContractBreak) []string {
	keys := sortedKeys(interactions)
	sort.SliceStable(keys, func(i, j int) bool {
		return keys[i] == "request" && keys[j] != "request"
	})
	return keys
}

func formatBreakLine(environment, interaction string, contractBreak ContractBreak) string {
	details := contractBreak.Details

	switch contractBreak.Reason {
	case "provider_resource_not_deployed_in_environment":
		line := fmt.Sprintf("provider is not deployed in %q", environment)
		if deployedIn, ok := details["deployedEnvironments"]; ok {
			line += fmt.Sprintf(" (deployed in: %s)", deployedIn)
		}
		return line
	case "provider_resource_not_found":
		return "no matching resource in provider"
	case "property_missing_in_provider":
		return fmt.Sprintf("property %q:%s required in %s (consumer) absent in %s (provider)",
			details["property"], details["propertyType"], details["consumerName"], details["providerName"])
	case "property_missing_in_consumer":
		return fmt.Sprintf("property %q:%s required in %s (provider) absent in %s (consumer)",
			details["property"], details["propertyType"], details["providerName"], details["consumerName"])
	case "property_optional_in_provider_required_in_consumer":
		return fmt.Sprintf("property %q:%s is optional in provider but required in consumer",
			details["property"], details["propertyType"])
	case "property_optional_in_consumer_required_in_provider":
		return fmt.Sprintf("property %q:%s is optional in consumer but required in provider",
			details["property"], details["propertyType"])
	case "property_type_mismatch":
		return fmt.Sprintf("property %q type mismatch — consumer has %s, provider has %s",
			details["property"], details["consumerPropertyType"], details["providerPropertyType"])
	default:
		return fallbackBreakLine(contractBreak.Reason, details)
	}
}

// fallbackBreakLine renders an unknown reason code verbatim with its details, so the
// CLI never swallows a break it does not have a template for.
func fallbackBreakLine(reason string, details map[string]string) string {
	if len(details) == 0 {
		return reason
	}

	keys := make([]string, 0, len(details))
	for key := range details {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, key+": "+details[key])
	}

	return fmt.Sprintf("%s (%s)", reason, strings.Join(pairs, ", "))
}
