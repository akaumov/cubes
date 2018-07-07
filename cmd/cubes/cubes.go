package main

import (
	"cubes/global"
	"cubes/instance"
	"fmt"
	"github.com/urfave/cli"
	"log"
	"os"
	"strconv"
	"strings"
)

func initProject(c *cli.Context) error {
	return nil
}

func main() {
	app := cli.NewApp()
	app.Commands = []cli.Command{
		{

			Name:   "init",
			Usage:  "init project",
			Action: initProject,
		},
		{
			Name:  "bus",
			Usage: "cubes bus",
			Subcommands: []cli.Command{
				{
					Name:   "start",
					Usage:  "start cubes bus",
					Action: startBus,
				},
			},
		},
		{
			Name:  "instance",
			Usage: "cube instance",
			Subcommands: []cli.Command{
				{
					Name:  "add",
					Usage: "adds cube instance",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "channels",
							Usage: "channels mapping: --channels 'cubeChannel1:handlerChannel1;cubeChannel2:handlerChannel2'",
						},
						cli.StringFlag{
							Name:  "ports",
							Usage: "ports mapping: --ports 'hostPort:handlerPort:protocol;80:8080:tcp'",
						},
						cli.StringFlag{
							Name:  "params",
							Usage: "params: --params 'param1:Value1;param2:Value2'",
						},
					},
					ArgsUsage: "[--ports] [--channels] [--params] name source",
					Action:    instanceAdd,
				},
				{
					Name:      "config",
					Usage:     "get cube instance config",
					ArgsUsage: "instanceName",
					Action:    instanceConfig,
				},
				{
					Name:      "remove",
					Usage:     "remove cube instance",
					ArgsUsage: "name",
					Action:    instanceRemove,
				},
				{
					Name:   "start",
					Usage:  "start cube instance",
					Action: instanceStart,
				},
				{
					Name:  "stop",
					Usage: "stops cube instance",
					Action: func(c *cli.Context) error {
						log.Println("stop instance")
						return nil
					},
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func parseChannelsMapping(channelsMappingRaw string) (*map[string]string, error) {
	channelsMapping := map[string]string{}

	if channelsMappingRaw != "" {

		for _, rawMap := range strings.Split(channelsMappingRaw, ";") {
			splittedMap := strings.Split(rawMap, ":")

			if len(splittedMap) != 2 {
				return nil, fmt.Errorf("Wrong channels mapping: %v\n", rawMap)
			}

			cubeChannel := splittedMap[0]
			handlerChannel := splittedMap[1]

			channelsMapping[handlerChannel] = cubeChannel
		}
	}

	return &channelsMapping, nil
}

func parsePortsMapping(portsMappingRaw string) (*[]instance.PortMap, error) {

	portsMapping := []instance.PortMap{}

	if portsMappingRaw != "" {

		for _, rawMap := range strings.Split(portsMappingRaw, ";") {
			splittedMap := strings.Split(rawMap, ":")

			if len(splittedMap) != 3 {
				return nil, fmt.Errorf("Wrong ports mapping: %v\n", rawMap)
			}

			hostPort, err := strconv.ParseUint(splittedMap[0], 10, 32)
			if err != nil {
				return nil, fmt.Errorf("Wrong host port format: %v/n", hostPort)
			}

			handlerPort, err := strconv.ParseUint(splittedMap[1], 10, 32)
			if err != nil {
				return nil, fmt.Errorf("Wrong cube port format: %v/n", handlerPort)
			}

			protocol := splittedMap[2]
			if protocol != "udp" && protocol != "tcp" {
				return nil, fmt.Errorf("Wrong port protocol: %v/n", protocol)
			}

			portsMapping = append(portsMapping, instance.PortMap{
				HostPort: uint(hostPort),
				CubePort: uint(handlerPort),
				Protocol: protocol,
			})
		}
	}

	return &portsMapping, nil
}

func parseInstanceParams(rawParams string) (*map[string]string, error) {

	params := map[string]string{}

	if rawParams != "" {

		for _, rawMap := range strings.Split(rawParams, ";") {
			splittedMap := strings.Split(rawMap, ":")

			if len(splittedMap) != 2 {
				return nil, fmt.Errorf("Wrong params format: %v\n", rawMap)
			}

			key := splittedMap[0]
			value := splittedMap[1]

			params[key] = value
		}
	}

	return &params, nil
}

func instanceAdd(c *cli.Context) error {
	args := c.Args()

	//TODO: add instance name format check
	name := args.Get(0)
	if name == "" {
		return fmt.Errorf("instance name is required")
	}

	source := args.Get(1)
	if source == "" {
		return fmt.Errorf("instance source is required")
	}

	channelsMappingRaw := c.String("channels")
	channelsMapping, err := parseChannelsMapping(channelsMappingRaw)
	if err != nil {
		return err
	}

	portsMappingRaw := c.String("ports")
	portsMapping, err := parsePortsMapping(portsMappingRaw)
	if err != nil {
		return err
	}

	paramsRaw := c.String("params")
	params, err := parseInstanceParams(paramsRaw)
	if err != nil {
		return err
	}

	err = instance.Add(
		name,
		source,
		*params,
		*portsMapping,
		*channelsMapping,
	)

	return err
}

func instanceConfig(c *cli.Context) error {
	args := c.Args()

	//TODO: add instance name format check
	name := args.Get(0)
	if name == "" {
		return fmt.Errorf("instance name is required")
	}

	config, err := instance.GetConfigText(name)
	fmt.Println(config)
	return err
}

func instanceRemove(c *cli.Context) error {
	args := c.Args()
	name := args.Get(0)

	if name == "" {
		return fmt.Errorf("instance name is required")
	}

	return instance.Remove(name)
}

func instanceStart(c *cli.Context) error {
	args := c.Args()
	name := args.Get(0)

	if name == "" {
		return fmt.Errorf("instance name is required")
	}

	return instance.Start(name)
}

func startBus(c *cli.Context) error {
	return global.StartBus()
}
