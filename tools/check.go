package tools

import (
	"context"
	"os"

	"github.com/dingdinglz/dingtools/dinglog"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func CheckContainer(cli *client.Client, ctx context.Context, logger *dinglog.DingLogger, ID string) bool {
	container_list, err := cli.ContainerList(ctx, types.ContainerListOptions{
		All: true,
	})
	if err != nil {
		logger.Error("获取容器列表错误", err.Error())
		os.Exit(1)
	}
	var have_container bool = false
	for _, i := range container_list {
		if i.ID == ID {
			have_container = true
		}
	}
	if !have_container {
		return false
	}
	image_list, err := cli.ImageList(ctx, types.ImageListOptions{
		All: true,
	})
	if err != nil {
		logger.Error("获取image列表错误", err.Error())
		os.Exit(1)
	}
	for _, i := range image_list {
		if i.RepoTags[0] == "codedocker:latest" {
			return true
		}
	}
	return false
}
