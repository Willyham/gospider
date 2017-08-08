// Copyright © 2017 Will Demaine
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"log"
	"os"

	"github.com/Willyham/gospider/spider"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the spider",
	RunE: func(cmd *cobra.Command, args []string) error {
		conf, err := NewConfig(viper.AllSettings())
		if err != nil {
			return err
		}

		spider := spider.New(
			spider.WithRoot(conf.RootURL),
			spider.WithIgnoreRobots(conf.IgnoreRobots),
		)

		err = spider.Run()
		if err != nil {
			log.Fatal("error running spider: ", err)
		}
		return spider.Report(os.Stdout)
	},
}

func init() {
	RootCmd.AddCommand(startCmd)

	startCmd.Flags().StringP("root", "r", "", "Root URL to spider from")
	startCmd.Flags().BoolP("ignore-robots", "i", false, "Ignore robots.txt")

	viper.BindPFlag("root", startCmd.Flags().Lookup("root"))
	viper.BindPFlag("ignore-robots", startCmd.Flags().Lookup("ignore-robots"))
}
