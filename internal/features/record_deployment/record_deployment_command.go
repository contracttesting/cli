package record_deployment

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/contracttesting/cli/internal/components"
	"github.com/spf13/cobra"
)

const requestTimeout = 30 * time.Second

func NewRecordDeploymentCommand(client *RecordDeploymentClient) *cobra.Command {
	commandHandler := func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true
		command.SilenceErrors = true

		participant := args[0]

		version, err := command.Flags().GetString("version")
		if err != nil {
			return fmt.Errorf("get version: %w", err)
		}

		environment, err := command.Flags().GetString("environment")
		if err != nil {
			return fmt.Errorf("get environment: %w", err)
		}

		forceRecord, err := command.Flags().GetBool("force-record")
		if err != nil {
			return fmt.Errorf("get force-record: %w", err)
		}

		ctx, cancel := context.WithTimeout(command.Context(), requestTimeout)
		defer cancel()

		requestBody := &RecordDeploymentRequestBody{
			Participant: participant,
			Version:     version,
			Environment: environment,
			Force:       forceRecord,
		}

		response, err := client.Record(ctx, requestBody)
		if err != nil {
			return err
		}

		switch response.Reason {
		case ReasonCompatibilityCheckRequired:
			fmt.Fprint(command.OutOrStdout(), formatCheckRequiredReport(participant, version, environment, response.Results))
			return components.ErrSilent
		case ReasonNotDeployable:
			fmt.Fprint(command.OutOrStdout(), formatNotDeployableReport(participant, version, environment, response.Results))
			return components.ErrSilent
		}

		fmt.Fprintf(command.OutOrStdout(), "🎉 %s\n", response.Message)
		return nil
	}

	command := &cobra.Command{
		Use:   "record-deployment [participant]",
		Short: "Record a deployment of a participant version to an environment",
		Args:  cobra.ExactArgs(1),
		RunE:  commandHandler,
	}

	command.Flags().String("version", "", "Deployed version, e.g. a commit hash or semver tag (required)")
	command.Flags().String("environment", "", "Target environment name (required)")
	command.Flags().Bool("force-record", false, "Record the deployment even when the compatibility verdict is not deployable")
	_ = command.MarkFlagRequired("version")
	_ = command.MarkFlagRequired("environment")

	return command
}

func formatCheckRequiredReport(participant, version, environment string, results map[string]RecordDeploymentResult) string {
	var report strings.Builder
	fmt.Fprintf(&report, "❌ %s %s cannot be recorded as deployed to %s: a fresh compatibility check is required\n",
		participant, version, environment)

	for _, name := range sortedResultKeys(results) {
		fmt.Fprintf(&report, "  - %s\n", formatDriftLine(name, results[name]))
	}

	fmt.Fprintf(&report, "\nRun: ctio can-i-deploy %s --version %s --environment %s\n", participant, version, environment)

	return report.String()
}

// formatDriftLine renders one drifted counterpart: what the last check saw
// against what the environment runs now.
func formatDriftLine(name string, result RecordDeploymentResult) string {
	switch {
	case result.CheckedVersion == nil && result.DeployedVersion != nil:
		return fmt.Sprintf("%s: checked as not deployed, but %s is deployed now", name, *result.DeployedVersion)
	case result.CheckedVersion != nil && result.DeployedVersion == nil:
		return fmt.Sprintf("%s: checked against %s, but it is no longer deployed", name, *result.CheckedVersion)
	case result.CheckedVersion != nil && result.DeployedVersion != nil:
		return fmt.Sprintf("%s: checked against %s, but %s is deployed now", name, *result.CheckedVersion, *result.DeployedVersion)
	default:
		return fmt.Sprintf("%s: never checked", name)
	}
}

func formatNotDeployableReport(participant, version, environment string, results map[string]RecordDeploymentResult) string {
	var report strings.Builder
	fmt.Fprintf(&report, "❌ %s %s is not deployable to %s\n", participant, version, environment)

	for _, name := range sortedResultKeys(results) {
		result := results[name]

		counterpart := name
		if result.CounterpartVersion != nil {
			counterpart = fmt.Sprintf("%s (%s)", name, *result.CounterpartVersion)
		}

		fmt.Fprintf(&report, "  - %s: %s\n", counterpart, formatFailureReason(environment, result.Reason))
	}

	fmt.Fprintf(&report, "\nRun: ctio can-i-deploy %s --version %s --environment %s for the full report\n",
		participant, version, environment)

	return report.String()
}

func formatFailureReason(environment, reason string) string {
	switch reason {
	case "provider_resource_not_deployed_in_environment":
		return fmt.Sprintf("provider is not deployed in %q", environment)
	case "provider_resource_not_found":
		return "no matching resource in provider"
	default:
		// e.g. "property_type_mismatch" → "property type mismatch"
		return strings.ReplaceAll(reason, "_", " ")
	}
}

func sortedResultKeys(results map[string]RecordDeploymentResult) []string {
	keys := make([]string, 0, len(results))
	for key := range results {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
