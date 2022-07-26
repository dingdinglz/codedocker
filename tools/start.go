package tools

import (
	"context"
	"os"
	"time"

	"github.com/dingdinglz/dingtools/dinglog"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func StartOrRestartConstainer(cli *client.Client, ctx context.Context, logger *dinglog.DingLogger, ID string) {
	info, err := cli.ContainerInspect(ctx, ID)
	if err != nil {
		logger.Error("获取容器信息失败！", err.Error())
		os.Exit(1)
	}
	if info.State.Running {
		stop_time := time.Duration(5 * time.Second)
		logger.Info("正在停止环境容器", ID)
		err := cli.ContainerStop(ctx, ID, &stop_time)
		if err != nil {
			logger.Error("停止容器错误！", err.Error())
		}
	}
	logger.Info("正在启动环境容器", ID)
	err = cli.ContainerStart(ctx, ID, types.ContainerStartOptions{})
	if err != nil {
		logger.Error("启动容器错误！", err.Error())
		os.Exit(1)
	}
}
