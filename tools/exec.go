package tools

import (
	"context"
	"os"

	"github.com/dingdinglz/dingtools/dinglog"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

var (
	exec_id string
)

func StartExec(ctx context.Context, cli *client.Client, logger *dinglog.DingLogger, container_id string, port string) {
	logger.Info("正在创建后端exec...")
	res_create, err := cli.ContainerExecCreate(ctx, container_id, types.ExecConfig{
		WorkingDir: "/base",
		Cmd:        []string{"/base/base", "--port", port},
	})
	if err != nil {
		logger.Error("创建exec失败！", err.Error())
		os.Exit(1)
	}
	exec_id = res_create.ID
	logger.Info("正在启动后端exec...")
	err = cli.ContainerExecStart(ctx, exec_id, types.ExecStartCheck{})
	if err != nil {
		logger.Error("启动后端exec错误！", err.Error())
		os.Exit(1)
	}
}

func CheckEvery(ctx context.Context, cli *client.Client, logger *dinglog.DingLogger, container_id string) int {
	res_container_info, err := cli.ContainerInspect(ctx, container_id)
	if err != nil {
		logger.Error("获取容器", container_id, "信息错误！", err.Error())
		os.Exit(1)
	}
	if !res_container_info.State.Running {
		return 1
	}
	res_exec_info, err := cli.ContainerExecInspect(ctx, exec_id)
	if err != nil {
		logger.Error("获取exec信息错误！", err.Error())
		os.Exit(1)
	}
	if !res_exec_info.Running {
		return 2
	}
	return 0
}
