/*
 * Copyright The Dragonfly Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package app

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/dragonflyoss/Dragonfly/common/util"
	"github.com/dragonflyoss/Dragonfly/dfdaemon"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/config"
	"github.com/dragonflyoss/Dragonfly/dfdaemon/constant"
	"github.com/dragonflyoss/Dragonfly/version"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v2"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:               "dfdaemon",
	Short:             "The dfdaemon is a proxy that intercepts image download requests.",
	Long:              "The dfdaemon is a proxy between container engine and registry used for pulling images.",
	DisableAutoGenTag: true, // disable displaying auto generation tag in cli docs
	RunE: func(cmd *cobra.Command, args []string) error {
		if viper.GetBool("version") {
			fmt.Println("dfdaemon version:", version.DFDaemonVersion)
			return nil
		}

		viper.SetConfigFile(viper.GetString("config"))
		viper.SetConfigType("yaml")
		if err := viper.ReadInConfig(); err != nil {
			return err
		}

		var cfg config.Properties

		exitOnError(
			viper.Unmarshal(&cfg, func(dc *mapstructure.DecoderConfig) {
				dc.TagName = "yaml"
				dc.DecodeHook = decodeWithYAML(
					reflect.TypeOf(config.Regexp{}),
					reflect.TypeOf(config.URL{}),
				)
			}),
			"unmarshal yaml",
		)

		initDfdaemon(cfg)

		if viper.GetBool("verbose") {
			yaml.NewEncoder(os.Stdout).Encode(cfg)
		}

		s, err := dfdaemon.NewFromConfig(cfg)
		if err != nil {
			return err
		}

		return s.Start()
	},
}

func init() {
	self, err := filepath.Abs(os.Args[0])
	exitOnError(err, "get exec path")
	defaultDfgetPath := filepath.Join(filepath.Dir(self), "dfget")

	rootCmd.Flags().BoolP("version", "v", false, "version")
	rootCmd.Flags().String("config", constant.DefaultConfigPath, "the path of dfdaemon's configuration file")

	rootCmd.Flags().Bool("verbose", false, "verbose")
	rootCmd.Flags().Int("maxprocs", 4, "the maximum number of CPUs that the dfdaemon can use")

	// http server config
	rootCmd.Flags().String("hostIp", "127.0.0.1", "dfdaemon host ip, default: 127.0.0.1")
	rootCmd.Flags().Uint("port", 65001, "dfdaemon will listen the port")
	rootCmd.Flags().String("certpem", "", "cert.pem file path")
	rootCmd.Flags().String("keypem", "", "key.pem file path")

	rootCmd.Flags().String("registry", "https://index.docker.io", "registry mirror url, which will override the registry mirror settings in the config file if presented")

	// dfget download config
	rootCmd.Flags().String("localrepo", filepath.Join(os.Getenv("HOME"), ".small-dragonfly/dfdaemon/data/"), "temp output dir of dfdaemon")
	rootCmd.Flags().String("callsystem", "com_ops_dragonfly", "caller name")
	rootCmd.Flags().String("dfpath", defaultDfgetPath, "dfget path")
	rootCmd.Flags().String("ratelimit", util.NetLimit(), "net speed limit,format:xxxM/K")
	rootCmd.Flags().String("urlfilter", "Signature&Expires&OSSAccessKeyId", "filter specified url fields")
	rootCmd.Flags().Bool("notbs", true, "not try back source to download if throw exception")
	rootCmd.Flags().StringSlice("node", nil, "specify the addresses(IP:port) of supernodes that will be passed to dfget.")

	viper.BindPFlags(rootCmd.Flags())
	viper.BindPFlag("registry_mirror.remote", rootCmd.Flag("registry"))
}

func exitOnError(err error, msg string) {
	if err != nil {
		logrus.Fatalf("%s: %v", msg, err)
	}
}

// Execute will process dfdaemon.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
}

func decodeWithYAML(types ...reflect.Type) mapstructure.DecodeHookFunc {
	return func(f, t reflect.Type, data interface{}) (interface{}, error) {
		for _, typ := range types {
			if t == typ {
				b, _ := yaml.Marshal(data)
				v := reflect.New(t)
				return v.Interface(), yaml.Unmarshal(b, v.Interface())
			}
		}
		return data, nil
	}
}
