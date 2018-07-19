package global

import (
	"github.com/akaumov/cubes/utils"
	"github.com/akaumov/cubes/instance"
	"github.com/akaumov/cube_executor"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	docker_client "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"golang.org/x/net/context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"os"
	"encoding/json"
	"io/ioutil"
)

const busImage = "nats"

type ProjectConfig struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type InstanceInfo struct {
	Status string                   `json:"status"`
	Config cube_executor.CubeConfig `json:"config"`
}

func getProjectConfigPath() (string, error) {
	currentDirectory, err := os.Getwd()
	if err != nil {
		return "", err
	}

	instanceConfigPath := filepath.Join(currentDirectory, "project.json")
	return instanceConfigPath, nil
}

func StartBus() error {
	log.Println("Running bus")

	err := utils.PullImage(busImage)
	if err != nil {
		return fmt.Errorf("can't run bus %v/n", err)
	}

	err = runBus()
	if err != nil {
		return fmt.Errorf("Can't run bus %v/n", err)
	}

	return nil
}

func runBus() error {
	config, err := GetConfig()
	if err != nil {
		return fmt.Errorf("can't read project config: %v", err)
	}

	ctx := context.Background()
	client, err := docker_client.NewEnvClient()

	if err != nil {
		log.Fatalf("can't connect to docker service:\n%v", err)
		return err
	}

	defer client.Close()

	resp, err := client.ContainerCreate(ctx, &container.Config{
		Image: busImage,
		Tty:   true,
		Cmd:   []string{"-p", "4444"},
		ExposedPorts: nat.PortSet{
			"4444/tcp": struct{}{},
		},
	}, &container.HostConfig{
		AutoRemove: true,
		NetworkMode: container.NetworkMode(config.Name + "_network"),
		PortBindings: nat.PortMap{
			"4444/tcp": []nat.PortBinding{
				{
					HostIP:   "",
					HostPort: "4444",
				},
			},
		},
	}, nil, "cubes-bus")

	if err != nil {
		log.Fatalf("can't create docker container:\n%v", err)
		return err
	}

	if err := client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		log.Fatalf("can't start  instance container:\n%v", err)
		return err
	}

	return nil
}

func GetConfigText() (string, error) {
	configPath, err := getProjectConfigPath()
	if err != nil {
		return "", nil
	}

	config, err := ioutil.ReadFile(configPath)
	return string(config), nil
}

func GetConfig() (*ProjectConfig, error) {
	rawConfig, err := GetConfigText()
	if err != nil {
		return nil, err
	}

	var config ProjectConfig
	err = json.Unmarshal(([]byte)(rawConfig), &config)

	if err != nil {
		return nil, fmt.Errorf("can't parse project config: %v/n", err)
	}

	return &config, nil
}

func InitProject(name string, description string) error {
	configPath, err := getProjectConfigPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(configPath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("project is already inited: %v/n", err)
		}
	}

	config, _ := json.MarshalIndent(ProjectConfig{
		Name: name,
		Description:description,
	}, "", "  ")

	err = ioutil.WriteFile(configPath, config, 0777)
	if err != nil {
		return err
	}

	return nil
}

func StartProject() error {
	err := CreatePrivateNetwork()
	if err != nil {
		return fmt.Errorf("can't create private network: %v", err)
	}

	err = runBus()
	if err != nil {
		return fmt.Errorf("can't start bus: %v", err)
	}

	return nil
}

func Status() error {
	return nil
}

func ProjectVersionLog() error {
	return nil
}

func CreatePrivateNetwork() error  {
	config, err := GetConfig()
	if err != nil {
		return fmt.Errorf("can't read project config: %v", err)
	}

	ctx := context.Background()
	client, err := docker_client.NewEnvClient()

	if err != nil {
		log.Fatalf("can't connect to docker service:\n%v", err)
		return err
	}

	defer client.Close()


	_, err = client.NetworkCreate(ctx, config.Name + "_network", types.NetworkCreate{
		Driver: "bridge",
	})

	return err
}

func GetListInstances() (*[]InstanceInfo, error) {
	instancesDirectoryPath, err := instance.GetInstancesDirectoryPath()
	if err != nil {
		return nil, err
	}

	configsPathPattern := filepath.Join(instancesDirectoryPath, "*.json")
	files, err := filepath.Glob(configsPathPattern)
	if err != nil {
		return nil, err
	}

	result := []InstanceInfo{}

	for _, configPath := range files {
		_, fileName := filepath.Split(configPath)
		instanceName := strings.TrimSuffix(fileName, ".json")

		config, err := instance.GetConfig(instanceName)
		if err != nil {
			return nil, fmt.Errorf("can't read instance config %v/n", err)
		}

		result = append(result, InstanceInfo{
			Config: *config,
		})
	}

	return &result, err
}
