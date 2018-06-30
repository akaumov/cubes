package cubes

import (
	"log"
	"github.com/docker/docker/api/types"
	docker_client "github.com/docker/docker/client"
	"io"
	"os"
	"golang.org/x/net/context"
	"github.com/docker/docker/api/types/container"
	"os/exec"
	"io/ioutil"
	"path/filepath"
	"strings"
)

const go_image = "golang:1.10-alpine"

type CubesServer struct {
}

func NewCubesServer() *CubesServer {
	return &CubesServer{

	}
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

func compileGo() error {
	ctx := context.Background()
	client, err := docker_client.NewEnvClient()

	if err != nil {
		log.Fatalf("can't connect to docker service:\n%v", err)
		return err
	}

	defer client.Close()

	resp, err := client.ContainerCreate(ctx, &container.Config{
		Image: go_image,
		Cmd:   []string{"go", "build -x -v -o ./build/cube ./cmd/cube"},
		Tty:   true,
	}, &container.HostConfig{
		AutoRemove: true,
	}, nil, "")


	if err != nil {
		log.Fatalf("can't create docker container:\n%v", err)
		return err
	}

	if err := client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		log.Fatalf("can't start docker container:\n%v", err)
		return err
	}

	return nil
}

func depEnsure() error {
	cmd := exec.Command("dep", "ensure")
	return cmd.Run()
}

func cloneGitRepository(repo string, outputDir string) error {
	cmd := exec.Command("git", "clone", repo, outputDir)
	return cmd.Run()
}

func replaceStringInFile(filePath string, oldValue string, newValue string) error  {
	read, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	newContents := strings.Replace(string(read), oldValue, newValue, -1)

	err = ioutil.WriteFile(filePath, []byte(newContents), 0)
	if err != nil {
		return err
	}

	return nil
}

func prepareSourceCode(handlerRepository string) (string, error) {
	tempDir, err := ioutil.TempDir("", "cubes_")
	if err != nil {
		log.Fatalf("Can't create temp directory %v/n", err)
		return "", err
	}

	log.Print("Temp directory is created: ", tempDir)

	os.Chdir(tempDir)

	//os.Remove(filepath.Join(tempDir))
	err = cloneGitRepository("https://github.com/akaumov/cube_executor.git", "cube")
	if err != nil {
		log.Fatalf("Can't clone executor repository %v/n", err)
		return "", err
	}

	err = replaceStringInFile(filepath.Join(tempDir, "cube", "cube.go"),"github.com/akaumov/echo-cube", handlerRepository)
	if err != nil {
		log.Fatalf("Can't set handler: %v/n", err)
		return "", err
	}

	return filepath.Join(tempDir, "cube"), nil
}

func (c *CubesServer) Start() {
	pullImage("golang:1.10-alpine")

	sourceCodePath, err := prepareSourceCode("github.com/akaumov/echo-cube")
	if err != nil {
		log.Fatalf("Can't clone executor repository %v/n", err)
	}

	//compileGo()
}
