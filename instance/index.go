package instance

import (
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"cubes/utils"
	"github.com/akaumov/cube_executor"
	docker_client "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"golang.org/x/net/context"
	"strconv"
)

const Version = "1"
const instancesDirectoryName = "instances"
const cubeCompilerImage = "azatk/cube-compiler:latest"
const cubeInstanceImage = "azatk/cube-instance:latest"

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

func Add(name string, source string, class string, queueGroup string, params map[string]string, portsMapping []cube_executor.PortMap, channelsMapping map[cube_executor.CubeChannel]cube_executor.BusChannel) error {
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

	config, _ := json.MarshalIndent(cube_executor.CubeConfig{
		SchemaVersion:     Version,
		Version:           "1",
		Name:              name,
		Source:            source,
		Class:             class,
		QueueGroup:        queueGroup,
		Params:            params,
		PortsMapping:      portsMapping,
		ChannelsMapping:   channelsMapping,
		NumberOfListeners: 1,
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

func GetConfigText(name string) (string, error) {
	instanceConfigPath, err := getInstanceConfigPath(name)
	if err != nil {
		return "", nil
	}

	instanceConfig, err := ioutil.ReadFile(instanceConfigPath)
	return string(instanceConfig), nil
}

func GetConfig(name string) (*cube_executor.CubeConfig, error) {
	rawConfig, err := GetConfigText(name)
	if err != nil {
		return nil, err
	}

	var config cube_executor.CubeConfig
	err = json.Unmarshal(([]byte)(rawConfig), &config)

	if err != nil {
		return nil, fmt.Errorf("can't parse instance config: %v/n", err)
	}

	return &config, nil
}

func Start(name string) error {
	instanceConfig, err := GetConfig(name)
	if err != nil {
		return err
	}

	log.Println("Pulling cube compiler image...")
	err = utils.PullImage(cubeCompilerImage)
	if err != nil {
		return fmt.Errorf("can't pull compiler image: %v/n", err)
	}

	log.Println("Compiling cube...")
	tempDir, err := ioutil.TempDir("", "cubes_")
	if err != nil {
		return fmt.Errorf("can't create temp directory for build %v/n", err)
	}

	defer func() { os.RemoveAll(tempDir) }()

	err = compileCube(instanceConfig.Source, tempDir)
	if err != nil {
		return fmt.Errorf("can't compile cube %v/n", err)
	}

	log.Println("Runing cube instance...")
	err = utils.PullImage(cubeInstanceImage)
	if err != nil {
		return fmt.Errorf("can't pull cube instance image: %v/n", err)
	}

	appPath := filepath.Join(tempDir, "cube.tar")
	configPath, err := getInstanceConfigPath(instanceConfig.Name)

	err = runCubeInstance(appPath, *instanceConfig, configPath)
	if err != nil {
		log.Fatalf("Can't run cube instance %v/n", err)
		panic(err)
	}

	return nil
}

func Stop(name string) error {
	return nil
}

func Ping(name string) error {
	return nil
}

func compileCube(cubePackage string, outputDir string) error {
	ctx := context.Background()
	client, err := docker_client.NewEnvClient()

	if err != nil {
		log.Fatalf("can't connect to docker service:\n%v", err)
		return err
	}

	defer client.Close()

	resp, err := client.ContainerCreate(ctx, &container.Config{
		Image: cubeCompilerImage,
		Tty:   true,
		Env:   []string{"CUBE_PACKAGE=" + cubePackage},
	}, &container.HostConfig{
		AutoRemove: true,
		Binds:      []string{outputDir + ":/build:rw"},
	}, nil, "")

	if err != nil {
		log.Fatalf("can't create docker container:\n%v", err)
		return err
	}

	if err := client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		log.Fatalf("can't start docker container:\n%v", err)
		return err
	}

	client.ContainerWait(ctx, resp.ID)
	return nil
}

func runCubeInstance(appPath string, config cube_executor.CubeConfig, configPath string) error {
	ctx := context.Background()
	client, err := docker_client.NewEnvClient()

	if err != nil {
		log.Fatalf("can't connect to docker service:\n%v", err)
		return err
	}

	defer client.Close()

	client.ContainerStop(ctx, config.Name, nil)
	client.ContainerRemove(ctx, config.Name, types.ContainerRemoveOptions{})

	exposedPorts := nat.PortSet{}
	portMap := nat.PortMap{}

	for _, portData := range config.PortsMapping {

		port, err := nat.NewPort(string(portData.Protocol), strconv.FormatUint(uint64(portData.CubePort), 10))
		if err != nil {
			return err
		}

		exposedPorts[port] = struct{}{}
		portMap[port] = []nat.PortBinding{
			{
				HostIP:   "",
				HostPort: strconv.FormatUint(uint64(portData.HostPort), 10),
			},
		}
	}

	resp, err := client.ContainerCreate(ctx, &container.Config{
		Image:        cubeInstanceImage,
		Tty:          true,
		ExposedPorts: exposedPorts,
		Labels: map[string]string{
			"_CUBE":             "true",
			"_CUBE_CLASS":       config.Class,
			"_CUBE_NAME":        config.Name,
			"_CUBE_VERSION":     config.Version,
			"_CUBE_QUEUE_GROUP": config.QueueGroup,
		},
	}, &container.HostConfig{
		AutoRemove:   true,
		Links:        []string{"cubes-bus:cubes-bus"},
		Binds:        []string{configPath + ":/config.json:rw"},
		PortBindings: portMap,
	}, nil, config.Name)

	if err != nil {
		log.Fatalf("can't create docker container:\n%v", err)
		return err
	}

	file, err := os.Open(appPath)
	if err != nil {
		log.Fatalf("can't read compiled cube:\n%v", err)
		return err
	}

	err = client.CopyToContainer(ctx, resp.ID, "/home/app", file, types.CopyToContainerOptions{
		AllowOverwriteDirWithFile: true,
	})

	if err != nil {
		log.Fatalf("can't copy compiled app to instance container:\n%v", err)
		return err
	}

	if err := client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		log.Fatalf("can't start  instance container:\n%v", err)
		return err
	}

	return nil
}
