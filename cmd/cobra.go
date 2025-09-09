package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/blocktransaction/zen/cmd/api"
	"github.com/spf13/cobra"
)

const (
	mainName = "zen"
	version  = " 1.0.0"
)

// cmd
var rootCmd = &cobra.Command{
	Use:          mainName,
	Short:        mainName,
	SilenceUsage: true,
	Long:         mainName,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			tip()
			return errors.New("requires at least one arg")
		}
		return nil
	},
	PersistentPreRunE: func(*cobra.Command, []string) error { return nil },
	Run: func(cmd *cobra.Command, args []string) {
		tip()
	},
}

// init
func init() {
	rootCmd.AddCommand(api.StartCmd, migrateCmd)
}

// 提示
func tip() {
	usageStr := `欢迎使用  ` + mainName + version + ` 可以使用 -h` + ` 查看命令`
	fmt.Printf("%s\n", usageStr)
}

// Execute : apply commands
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}
