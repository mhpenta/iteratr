package main

import (
	"fmt"
	"os"

	"github.com/mark3labs/iteratr/internal/template"
	"github.com/spf13/cobra"
)

var genTemplateFlags struct {
	output string
}

var genTemplateCmd = &cobra.Command{
	Use:   "gen-template",
	Short: "Export the default prompt template",
	Long: `Export the default prompt template to a file.

The generated template can be customized and then used with the --template flag
in the build command. Templates use {{variable}} syntax for substitution.`,
	RunE: runGenTemplate,
}

func init() {
	genTemplateCmd.Flags().StringVarP(&genTemplateFlags.output, "output", "o", ".iteratr.template", "Output file")
}

func runGenTemplate(cmd *cobra.Command, args []string) error {
	// Get default template content
	content := template.DefaultTemplate

	// Write to file
	if err := os.WriteFile(genTemplateFlags.output, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write template: %w", err)
	}

	fmt.Printf("Template exported to: %s\n", genTemplateFlags.output)
	return nil
}
