package utils

import (
	"github.com/docker/docker/api/types"
	docker_client "github.com/docker/docker/client"
	"golang.org/x/net/context"
	"io"
	"log"
	"os"
)

func PullImage(image string) error {
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
