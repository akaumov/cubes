package cubes

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	docker_client "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"golang.org/x/net/context"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

const cubeCompilerImage = "azatk/cube-compiler:latest"
const cubeInstanceImage = "azatk/cube-instance:latest"
const busImage = "nats"

type CubesServer struct {
}

func NewCubesServer() *CubesServer {
	return &CubesServer{}
}

func pullImage(image string) error {
	ctx := context.Background()
	client, err := docker_client.NewEnvClient()

	if err != nil {
		log.Fatalf("can't connect to docker service:\n%v", err)
		return err
	}

	out, err := client.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		return err
	}

	defer out.Close()
	defer client.Close()

	io.Copy(os.Stdout, out)

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
		//AutoRemove: true,
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

func runCubeInstance(appPath string, instanceName string) error {
	ctx := context.Background()
	client, err := docker_client.NewEnvClient()

	if err != nil {
		log.Fatalf("can't connect to docker service:\n%v", err)
		return err
	}

	defer client.Close()

	client.ContainerStop(ctx, instanceName, nil)
	client.ContainerRemove(ctx, instanceName, types.ContainerRemoveOptions{})

	resp, err := client.ContainerCreate(ctx, &container.Config{
		Image: cubeInstanceImage,
		Tty:   true,
	}, &container.HostConfig{
		//AutoRemove: true,
		Links: []string{"cubes-bus:cubes-bus"},
	}, nil, instanceName)

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

func (c *CubesServer) Start() {

	log.Println("Running bus")
	pullImage(busImage)
	err := runBus()
	if err != nil {
		log.Fatalf("Can't run bus %v/n", err)
		panic(err)
	}

	log.Println("Compile cube")
	pullImage(cubeCompilerImage)
	tempDir, err := ioutil.TempDir("", "cubes_")
	if err != nil {
		log.Fatalf("Can't create temp directory for build %v/n", err)
		panic(err)
	}

	err = compileCube("github.com/akaumov/cube-http-gateway", tempDir)
	if err != nil {
		log.Fatalf("Can't compile cube %v/n", err)
		panic(err)
	}

	log.Println("Run cube instance")
	pullImage(cubeInstanceImage)

	err = runCubeInstance(filepath.Join(tempDir, "cube.tar"), "gateway")
	if err != nil {
		log.Fatalf("Can't run cube instance %v/n", err)
		panic(err)
	}
}
