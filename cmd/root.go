/*
Copyright © 2020 Radio Bern RaBe - Lucas Bickel <hairmare@rabe.ch>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"

	"github.com/radiorabe/virtual-saemubox/box"
)

var cfgFile string
var target string
var pathfinder string
var pathfinderAuth string
var device string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "virtual-saemubox",
	Short: "Emulates a Sämubox with data queried from Pathfinder",
	Long: `This subscribes to pin state data on Pathfinder and publishes that info in the
legacy UDP telnet format that quite a few of our legacy apps still expect.

It is a stop-gap measure that helps us migrate to Pathfinder ASAP enabling us to
refactor legacy apps at a later point.`,
	Run: func(cmd *cobra.Command, args []string) {
		box.Execute(target, pathfinder, pathfinderAuth, device)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.virtual-saemubox.yaml)")

	rootCmd.PersistentFlags().StringVar(&target, "target", "app:4001", "Target host:port")
	rootCmd.PersistentFlags().StringVar(&pathfinder, "pathfinder", "pathfinder-01.service.int.rabe.ch:9600", "Pathfinder host:port")
	rootCmd.PersistentFlags().StringVar(&device, "device", "Devices#0.WinDriver#[tcp://10.130.226.7:93].LwrpInterpreter#0.LwrpRoot#0.Gpi#1.GpioPinState#1 PinState", "Pathfinder endpoint to sub to")
	rootCmd.PersistentFlags().StringVar(&pathfinderAuth, "pathfinder-auth", "Admin Admin", "Pathfinder user pass")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".virtual-saemubox" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".virtual-saemubox")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
