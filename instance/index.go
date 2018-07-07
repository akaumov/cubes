package instance

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

const Version = "1"
const instancesDirectoryName = "instances"

type PortMap struct {
	CubePort uint   `json:"cubePort"`
	HostPort uint   `json:"hostPort"`
	Protocol string `json:"protocol"`
}

type InstanceConfig struct {
	SchemaVersion   string            `json:"schemaVersion"`
	Version         string            `json:"version"`
	Name            string            `json:"name"`
	Source          string            `json:"source"`
	Params          map[string]string `json:"params"`
	PortsMapping    []PortMap         `json:"portsMapping"`
	ChannelsMapping map[string]string `json:"channelsMapping"`
}

func getInstancesDirectoryPath() (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	instancesDirectory := filepath.Join(pwd, instancesDirectoryName)
	return instancesDirectory, nil
}

func getInstanceConfigPath(name string) (string, error) {
	instancesDirectory, err := getInstancesDirectoryPath()
	if err != nil {
		return "", err
	}

	instanceConfigPath := filepath.Join(instancesDirectory, name+".json")
	return instanceConfigPath, nil
}

func Add(name string, source string, params map[string]string, portsMapping []PortMap, channelsMapping map[string]string) error {
	instancesDirectory, err := getInstancesDirectoryPath()
	if err != nil {
		return err
	}

	//TODO: add checking usage of instance name
	if _, err := os.Stat(instancesDirectory); err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		err = os.Mkdir(instancesDirectory, 0777)
		if err != nil {
			return err
		}
	}

	instanceFile, err := getInstanceConfigPath(name)
	if err != nil {
		return err
	}

	if _, err := os.Stat(instanceFile); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("instance file is already exist: %v/n", err)
		}
	}

	config, _ := json.MarshalIndent(InstanceConfig{
		SchemaVersion:   Version,
		Version:         "1",
		Name:            name,
		Source:          source,
		Params:          params,
		PortsMapping:    portsMapping,
		ChannelsMapping: channelsMapping,
	}, "", "  ")

	err = ioutil.WriteFile(instanceFile, config, 0777)
	if err != nil {
		return err
	}

	return nil
}

func Remove(name string) error {
	//TODO: check instance state
	instanceConfigPath, err := getInstanceConfigPath(name)
	if err != nil {
		return err
	}

	if _, err := os.Stat(instanceConfigPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("instance file is not exist: %v", err)
		}

		return err
	}

	return os.Remove(instanceConfigPath)
}

func GetConfig(name string) (string, error) {
	instanceConfigPath, err := getInstanceConfigPath(name)
	if err != nil {
		return "", nil
	}

	instanceConfig, err := ioutil.ReadFile(instanceConfigPath)
	return string(instanceConfig), nil
}

func Start(name string) error {
	return nil
}

func Stop(name string) error {
	return nil
}

func Ping(name string) error {
	return nil
}
