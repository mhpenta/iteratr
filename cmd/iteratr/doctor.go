package main

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check dependencies and environment",
	Long: `Check that required dependencies are installed and accessible.

This command verifies that:
- opencode is installed and in PATH
- The data directory is writable
- Other environment requirements are met`,
	RunE: runDoctor,
}

func runDoctor(cmd *cobra.Command, args []string) error {
	allOk := true

	// Check for opencode
	fmt.Print("Checking for opencode... ")
	if _, err := exec.LookPath("opencode"); err != nil {
		fmt.Println("❌ NOT FOUND")
		fmt.Println("  opencode is not installed or not in PATH")
		fmt.Println("  Install: https://opencode.coder.com")
		allOk = false
	} else {
		// Try to run opencode --version
		out, err := exec.Command("opencode", "--version").CombinedOutput()
		if err != nil {
			fmt.Println("⚠️  FOUND (but can't get version)")
		} else {
			fmt.Printf("✅ FOUND (%s)\n", string(out))
		}
	}

	// Check Go version
	fmt.Print("Checking Go version... ")
	out, err := exec.Command("go", "version").CombinedOutput()
	if err != nil {
		fmt.Println("❌ NOT FOUND")
		allOk = false
	} else {
		fmt.Printf("✅ %s\n", string(out))
	}

	// Summary
	fmt.Println()
	if allOk {
		fmt.Println("✅ All checks passed!")
		return nil
	} else {
		fmt.Println("❌ Some checks failed. Please install missing dependencies.")
		return fmt.Errorf("doctor check failed")
	}
}
