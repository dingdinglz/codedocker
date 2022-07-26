package tools

import (
	"context"
	"os"
	"time"

	"github.com/dingdinglz/dingtools/dinglog"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func CleanContainerAndImage(cli *client.Client, ctx context.Context, logger *dinglog.DingLogger) {
	container_list, err := cli.ContainerList(ctx, types.ContainerListOptions{
		All: true,
	})
	if err != nil {
		logger.Error("获取容器列表错误", err.Error())
		os.Exit(1)
	}
	for _, i := range container_list {
		res_i, _, err := cli.ImageInspectWithRaw(ctx, i.ImageID)
		if err != nil {
			logger.Error("取父image错误！", err.Error())
			os.Exit(1)
		}
		logger.Debug(i.Names, i.ID, res_i.RepoTags)
		if i.Names[0] == "/codedockerC" || len(res_i.RepoTags) == 0 {
			logger.Info("正在清理容器", i.ID)
			i_stat, err := cli.ContainerInspect(ctx, i.ID)
			if err != nil {
				logger.Error("获取容器信息错误！", err.Error())
				os.Exit(1)
			}
			if i_stat.State.Running {
				stop := time.Duration(time.Second * 5)
				logger.Info("正在停止容器", i.ID)
				err := cli.ContainerStop(ctx, i.ID, &stop)
				if err != nil {
					logger.Error("停止容器错误！", err.Error())
					os.Exit(1)
				}
			}
			err = cli.ContainerRemove(ctx, i.ID, types.ContainerRemoveOptions{})
			if err != nil {
				logger.Error("删除容器错误！", err.Error())
				os.Exit(1)
			}
		}
	}

	image_list, err := cli.ImageList(ctx, types.ImageListOptions{
		All: true,
	})
	if err != nil {
		logger.Error("获取image列表错误", err.Error())
		os.Exit(1)
	}
	for _, i := range image_list {
		logger.Debug(i.RepoTags, i.ID)
		if i.RepoTags[0] == "codedocker:latest" || i.RepoTags[0] == "<none>:<none>" {
			logger.Info("正在清理image", i.ID)
			_, err := cli.ImageRemove(ctx, i.ID, types.ImageRemoveOptions{})
			if err != nil {
				logger.Error("删除image错误", err.Error())
				os.Exit(1)
			}
		}
	}
}
