package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"

	"github.com/jodydadescott/prisma-microseg-linter/example"
	"github.com/jodydadescott/prisma-microseg-linter/processor"
)

var (
	sanatize, validate, verbose bool
)

var rootCmd = &cobra.Command{}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "runs the processor",
	RunE: func(cmd *cobra.Command, args []string) error {

		log.SetOutput(os.Stdout)
		log.SetFlags(log.Lshortfile)

		if len(args) != 1 {
			return fmt.Errorf("Missing injest directory")
		}

		if !sanatize && !validate {
			return fmt.Errorf("nothing to do; consider --sanatize and/or --validate")
		}

		processor, err := processor.NewNamespace(args[0], verbose)
		if err != nil {
			log.Fatal(err)
		}

		var errors *multierror.Error

		if sanatize {
			err = processor.Sanatize()
			if err != nil {
				errors = multierror.Append(errors, err)
			}
		}

		if validate {
			err = processor.Validate()
			if err != nil {
				errors = multierror.Append(errors, err)
			}
		}

		if errors.ErrorOrNil() != nil {
			log.Fatal(errors.ErrorOrNil())
		}

		return nil
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "generates example config",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("Missing output directory")
		}
		return example.Write(args[0])
	},
}

// Execute executes the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	runCmd.PersistentFlags().BoolVar(&sanatize, "sanatize", false, "sanatizes config")
	runCmd.PersistentFlags().BoolVar(&validate, "validate", false, "validates config")
	runCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose logging")
	rootCmd.AddCommand(runCmd, configCmd)
}
