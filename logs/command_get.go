package logs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/solarwinds/swo-cli/swosdkgo"
	"github.com/solarwinds/swo-cli/swosdkgo/models/components"
	"github.com/solarwinds/swo-cli/swosdkgo/models/operations"
	"github.com/urfave/cli/v2"
	"strings"
	"time"
)

var flagsGet = []cli.Flag{
	&cli.StringFlag{Name: "group", Aliases: []string{"g"}, Usage: "group name to search"},
	&cli.StringFlag{Name: "min-time", Usage: "earliest time to search from", Value: "1 hour ago"},
	&cli.StringFlag{Name: "max-time", Usage: "latest time to search from"},
	&cli.StringFlag{Name: "system", Aliases: []string{"s"}, Usage: "system to search"},
	&cli.BoolFlag{Name: "json", Aliases: []string{"j"}, Usage: "output raw JSON", Value: false},
	&cli.BoolFlag{Name: "follow", Aliases: []string{"f"}, Usage: "enable live tailing", Value: false},
}

func runGet(cCtx *cli.Context) error {
	config, err := configure(cCtx)
	sdk := swosdkgo.New(
		swosdkgo.WithSecurity(config.Token),
		swosdkgo.WithServerURL(config.APIURL))

	req := operations.SearchLogsRequest{}

	filter := strings.Join(cCtx.Args().Slice(), " ")
	fmt.Println(filter)
	if filter != "" {
		req.Filter = swosdkgo.String(filter)
	}

	group := cCtx.String("group")
	if group != "" {
		req.Group = swosdkgo.String(group)
	}

	system := cCtx.String("system")
	if system != "" {
		filter = fmt.Sprintf("%s host:%s", filter, system)
		req.Filter = swosdkgo.String(filter)
	}

	maxTime := cCtx.String("max-time")
	if maxTime != "" {
		result, err := parseTime(maxTime)
		if err != nil {
			return errors.Join(errMaxTimeFlag, err)
		}

		req.EndTime = swosdkgo.String(result)
	}

	minTime := cCtx.String("min-time")
	if minTime != "" {
		result, err := parseTime(minTime)
		if err != nil {
			return errors.Join(errMinTimeFlag, err)
		}

		req.StartTime = swosdkgo.String(result)
	}

	follow := cCtx.Bool("follow")
	if follow {
		result, err := parseTime(time.Now().Add(-10 * time.Second).Format(time.RFC3339))
		if err != nil {
			return errors.Join(errMinTimeFlag, err)
		}

		req.StartTime = &result
		req.Direction = swosdkgo.String("tail")
	}

	jsonOut := cCtx.Bool("json")

	resp, err := sdk.Logs.SearchLogs(context.Background(), req)
	if err != nil {
		return err
	}

	if resp.Object != nil {
		for {
			logEvents := resp.GetObject().GetLogs()
			if err := printResult(logEvents, jsonOut); err != nil {
				return err
			}

			pageInfo := resp.GetObject().GetPageInfo()
			if pageInfo.GetNextPage() == "" {
				break
			}

			if follow && len(resp.GetObject().GetLogs()) == 0 {
				time.Sleep(2 * time.Second)
			}

			resp, err = resp.Next()
			if err != nil {
				return err
			}

			if resp == nil {
				break
			}
		}
	}

	return nil
}

func NewGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "command-line search for SolarWinds Observability log management service",
		Flags: flagsGet,
		ArgsUsage: `

EXAMPLES:
   swo logs get something
   swo logs get 1.2.3 Failure
   swo logs get -s ns1 "connection refused"
   swo logs get -f "(www OR db) (nginx OR pgsql) -accepted"
   swo logs get -f -g <SWO_GROUP_NAME> "(nginx OR pgsql) -accepted"
   swo logs get --min-time 'yesterday at noon' --max-time 'today at 4am' -g <SWO_GROUP_NAME>
   swo logs get -- -redis
`,
		Action: runGet,
	}
}

func printResult(logs []components.LogsEvent, jsonOut bool) error {
	for _, l := range logs {
		if jsonOut {
			log, err := json.Marshal(l)
			if err != nil {
				return err
			}

			fmt.Println(string(log))
		} else {
			t, err := time.Parse(time.RFC3339, l.Time)
			if err != nil {
				return err
			}

			if _, err = fmt.Printf("%s %s %s %s\n", t.Format("Jan 02 15:04:05"), l.Hostname, l.Program, l.Message); err != nil {
				return err
			}
		}
	}

	return nil
}
