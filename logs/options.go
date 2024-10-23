package logs

import (
	"errors"
	"fmt"
	"github.com/olebedev/when"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrInvalidDateTime = errors.New("Could not parse timestamp")

	now = time.Now()

	errMinTimeFlag  = errors.New("failed to parse --min-time flag")
	errMaxTimeFlag  = errors.New("failed to parse --max-time flag")
	errMissingToken = errors.New("failed to find token")

	timeLayouts = []string{
		time.Layout,
		time.ANSIC,
		time.UnixDate,
		time.RubyDate,
		time.RFC822,
		time.RFC822Z,
		time.RFC850,
		time.RFC1123,
		time.RFC1123Z,
		time.RFC3339,
		time.RFC3339Nano,
		time.Kitchen,
		time.Stamp,
		time.StampMilli,
		time.StampNano,
		time.DateTime,
		time.DateOnly,
		time.TimeOnly,
		"2006-01-02 15:04:05",
	}
)

type Configuration struct {
	APIURL string `yaml:"api-url"`
	Token  string `yaml:"token"`
}

func configure(cCtx *cli.Context) (*Configuration, error) {
	// TODO: We should refactor these references to cCtx into some kind of configuration package
	// 	that creates a configuration object of some kind.
	configPath := cCtx.String("config")
	if strings.HasPrefix(configPath, "~/") {
		usr, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("error while resolving current user to read configuration file: %w", err)
		}

		dir, filename := filepath.Split(configPath)
		dir = strings.TrimLeft(dir, "~/")
		configPath = filepath.Join(usr.HomeDir, dir, filename)
	}

	configuration := &Configuration{}
	if content, err := os.ReadFile(configPath); err == nil {
		err = yaml.Unmarshal(content, configuration)
		if err != nil {
			return nil, fmt.Errorf("error while unmarshaling %s config file: %w", configPath, err)
		}
	}

	if token := os.Getenv("SWO_API_TOKEN"); token != "" {
		configuration.Token = token
	}

	if configuration.Token == "" {
		return nil, errMissingToken
	}

	configuration.APIURL = cCtx.String("api-url")
	if apiUrl := os.Getenv("SWO_API_URL"); apiUrl != "" {
		configuration.APIURL = apiUrl
	}

	return configuration, nil
}

func parseTime(input string) (string, error) {
	location := time.Local
	if strings.HasSuffix(input, " UTC") {
		l, err := time.LoadLocation("UTC")
		if err != nil {
			return "", err
		}

		location = l

		input = strings.ReplaceAll(input, " UTC", "")
	}

	for _, layout := range timeLayouts {
		result, err := time.Parse(layout, input)
		if err == nil {
			result = result.In(location)
			return result.Format(time.RFC3339), nil
		}
	}

	result, err := when.EN.Parse(input, now)
	if err != nil {
		return "", err
	}

	if result == nil {
		return "", ErrInvalidDateTime
	}

	return result.Time.In(location).Format(time.RFC3339), nil
}
