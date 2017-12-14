package config

import (
	"os"
	"strings"

	"github.com/juju/loggo"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/ExpressenAB/bigip_exporter/collector"
	"github.com/pr8kerl/f5er/f5"
	"fmt"
)

type bigipEnvConfig struct {
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	BasicAuth bool   `yaml:"basic_auth"`
	Partitions  string `yaml:"partitions"`
}

type exporterConfig struct {
	BindAddress string `yaml:"bind_address"`
	BindPort    int    `yaml:"bind_port"`
	Config      string `yaml:"config"`
	Namespace   string `yaml:"namespace"`
	LogLevel    string `yaml:"log_level"`
}

// Config is a container for settings modifiable by the user
type Config struct {
	Module   map[string]bigipEnvConfig `yaml:"module"`
	Exporter exporterConfig            `yaml:"exporter"`
}

var (
	logger = loggo.GetLogger("")
)

func init() {
	loggo.ConfigureLoggers("<root>=INFO")
	registerFlags()
	bindFlags()
	bindEnvs()
	flag.Parse()

	if viper.GetString("exporter.config") != "" {
		readConfigFile(viper.GetString("exporter.config"))
	}

	logLevel := viper.GetString("exporter.log_level")

	if _, validLevel := loggo.ParseLevel(logLevel); validLevel {
		loggo.ConfigureLoggers("<root>=" + strings.ToUpper(logLevel))
		return
	}

	logger.Warningf("Invalid log level - Using info")
}

func registerFlags() {
	flag.Bool("bigip.basic_auth", false, "Use HTTP Basic authentication")
	flag.String("bigip.host", "localhost", "The host on which f5 resides")
	flag.Int("bigip.port", 443, "The port which f5 listens to")
	flag.String("bigip.username", "user", "Username")
	flag.String("bigip.password", "pass", "Password")
	flag.String("exporter.bind_address", "localhost", "Exporter bind address")
	flag.Int("exporter.bind_port", 9142, "Exporter bind port")
	flag.String("exporter.partitions", "", "A comma separated list of partitions which to export. (default: all)")
	flag.String("exporter.config", "", "bigip_exporter configuration file name.")
	flag.String("exporter.namespace", "bigip", "bigip_exporter namespace.")
	flag.String("exporter.log_level", "info", "Available options are trace, debug, info, warning, error and critical")
}

func bindFlags() {
	flag.VisitAll(func(f *flag.Flag) {
		err := viper.BindPFlag(f.Name, f)
		if err != nil {
			logger.Warningf("Failed to bind flag (%s)", err)
		}
	})
}

func bindEnvs() {
	viper.SetEnvPrefix("be")
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	flag.VisitAll(func(f *flag.Flag) {
		err := viper.BindEnv(f.Name)
		if err != nil {
			logger.Warningf("Failed to bind environment variable BE_%s (%s)", strings.ToUpper(strings.Replace(f.Name, ".", "_", -1)), err)
		}
	})
}

func readConfigFile(fileName string) {
	file, err := os.Open(fileName)
	if err != nil {
		logger.Warningf("Failed to open configuration file (%s)", err)
		return
	}
	viper.SetConfigType("yaml")
	err = viper.ReadConfig(file)
	if err != nil {
		logger.Warningf("Failed to read configuration file (%s)", err)
	}
}

// GetConfig returns an instance of Config containing the resulting parameters
// to the program
func GetConfig() *Config {
	c := Config{}
	list := viper.GetStringMap("configs")

	c.Module = make(map[string]bigipEnvConfig)
	for env := range list {
		c.Module[env] = bigipEnvConfig{
			Username: 	viper.GetString(fmt.Sprintf("configs.%s.username", env)),
			Password: 	viper.GetString(fmt.Sprintf("configs.%s.password", env)),
			BasicAuth: 	viper.GetBool(fmt.Sprintf("configs.%s.basic_auth", env)),
			Partitions: viper.GetString(fmt.Sprintf("configs.%s.partitions", env)),
			}
	}
	c.Exporter = exporterConfig{
		viper.GetString("exporter.bind_address"),
		viper.GetInt("exporter.bind_port"),
		viper.GetString("bigip_exporter.config"),
		viper.GetString("exporter.namespace"),
		viper.GetString("exporter.log_level"),
	}
	//logger.Infof("Config: [%v]", c) // if uncommented, will print password. Instead use c.String()
	return &c
}

func (c Config) CreateBigipCollector(bigipEndpoint string, moduleName string) (collector.BigipCollector, bool) {
	module, ok :=  c.Module[moduleName]
	if !ok {
		logger.Debugf("[%s] module not found.", moduleName)
		return collector.BigipCollector{}, false
	}else {
		var exporterPartitionsList []string
		if module.Partitions != "" {
			exporterPartitionsList = strings.Split(module.Partitions, ",")
		} else {
			exporterPartitionsList = nil
		}
		authMethod := f5.TOKEN
		if module.BasicAuth {
			authMethod = f5.BASIC_AUTH
		}
		bigip := f5.New(bigipEndpoint,module.Username,module.Password,authMethod)
		bigipCollector, err := collector.NewBigipCollector(bigip, c.Exporter.Namespace, exporterPartitionsList)
		if err != nil {
			logger.Errorf("When creating a new F5 colllector, an error occured. Error [%v]", err)
			return *bigipCollector, false
		}
			return *bigipCollector, true
	}
}

func (c Config) String() string {
	stringBuilder := "["
	for env := range c.Module {
		username := c.Module[env].Username
		basic_auth := c.Module[env].BasicAuth
		partitions := c.Module[env].Partitions
		stringBuilder += fmt.Sprintf("{%s:[username: %s, basic_auth: %s, partitions: %s]},", env, username, basic_auth, partitions)
	}
	stringBuilder += fmt.Sprintf("Expoerter: %s", c.Exporter)
	return stringBuilder
}