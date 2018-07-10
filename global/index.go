package global

import (
	"cubes/utils"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	docker_client "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"golang.org/x/net/context"
	"log"
	"path/filepath"
	"github.com/akaumov/cube_executor"
	"strings"
	"cubes/instance"
)

const busImage = "nats"

type InstanceInfo struct {
	Status string                   `json:"status"`
	Config cube_executor.CubeConfig `json:"config"`
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

func InitProject(name string, description string) error {
	return nil
}

func StartProject(name string) {
}

func Status() error {
	return nil
}

func ProjectVersionLog() error {
	return nil
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
