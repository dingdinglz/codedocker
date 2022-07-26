package main

import (
	"bufio"
	"context"
	"flag"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"codedocker/tools"

	"github.com/buger/jsonparser"
	"github.com/dingdinglz/dingtools/dingdb/dingnuts"
	"github.com/dingdinglz/dingtools/dinglog"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/gin-gonic/gin"
)

var (
	DockerClient  *client.Client
	UnDefault     *bool
	RunPath       string
	ContainerPort *string
	ContainerID   string
	SettingDB     *dingnuts.DingNuts
	Debug         *bool
	CleanAll      *bool
	Port          *string
)

func init() {
	UnDefault = flag.Bool("undefault", false, "使用非本机的docker或自定义的docker api")
	ContainerPort = flag.String("ContainerPort", "7000", "环境容器使用的端口")
	Debug = flag.Bool("debug", false, "Debug Mode")
	CleanAll = flag.Bool("clean", false, "清理所有数据，下次启动将重新初始化")
	Port = flag.String("port", "80", "控制端服务器端口")
	flag.Parse()
}

func main() {
	RunPath, _ = os.Getwd()
	logger := dinglog.NewLogger()
	if *Debug {
		logger.SetLevel(dinglog.Level_Debug)
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	logger.Info("正在连接docker api")
	if !*UnDefault {
		var err error
		DockerClient, err = client.NewClientWithOpts(client.FromEnv)
		if err != nil {
			logger.Error("连接docker api失败", err.Error())
			os.Exit(1)
		}
	}

	logger.Info("docker api version:", DockerClient.ClientVersion())

	if *CleanAll {
		tools.CleanContainerAndImage(DockerClient, context.Background(), logger)
		err := os.RemoveAll(filepath.Join(RunPath, "setting"))
		if err != nil {
			logger.Error("删除setting文件夹错误！", err.Error())
			os.Exit(1)
		}
		logger.Info("清理完毕！")
		os.Exit(0)
	}

	if !tools.PathExists(filepath.Join(RunPath, "setting")) {
		logger.Info("正在清理old image and container...")
		tools.CleanContainerAndImage(DockerClient, context.Background(), logger)
		tools.BuildDockerFileTar(logger, RunPath)
		logger.Info("正在初始化docker image...")
		clean_res, err := DockerClient.BuildCachePrune(context.Background(), types.BuildCachePruneOptions{
			All: true,
		})
		if err != nil {
			logger.Error("清理构建缓存失败！", err.Error())
			os.Exit(1)
		}
		logger.Info("共清理缓存", clean_res.SpaceReclaimed)
		i, _ := os.Open(filepath.Join(RunPath, "docker", "Dockerfile.tar.gz"))
		res, err := DockerClient.ImageBuild(context.Background(), i, types.ImageBuildOptions{
			Tags:    []string{"codedocker"},
			NoCache: true,
		})
		if err != nil {
			logger.Error("编译image失败", err.Error())
			os.Exit(1)
		}
		reader_build := bufio.NewReader(res.Body)
		for {
			line, err_read := reader_build.ReadString('\n')
			if err_read != nil {
				break
			}
			_, _, _, have := jsonparser.Get([]byte(line), "stream")
			if have == nil {
				stream_get, _ := jsonparser.GetString([]byte(line), "stream")
				logger.Info("[build]", stream_get)
				continue
			}
			_, _, _, have = jsonparser.Get([]byte(line), "error")
			if have == nil {
				error_get, _ := jsonparser.GetString([]byte(line), "error")
				logger.Error("image 构建失败！", error_get)
				os.Exit(1)
			}
		}
		logger.Info("image codedocker构建成功！")
		logger.Info("正在构建环境容器...")
		port, _ := nat.NewPort("tcp", *ContainerPort)
		res_create, err := DockerClient.ContainerCreate(context.Background(), &container.Config{
			Image:        "codedocker",
			Tty:          true,
			ExposedPorts: nat.PortSet{port: struct{}{}},
		}, &container.HostConfig{
			PortBindings: nat.PortMap{port: []nat.PortBinding{
				{HostIP: "0.0.0.0", HostPort: string(port)},
			}},
		}, nil, nil, "codedockerC")
		if err != nil {
			logger.Error("创建环境容器错误！", err.Error())
			os.Exit(1)
		}
		ContainerID = res_create.ID
		err = DockerClient.ContainerStart(context.Background(), res_create.ID, types.ContainerStartOptions{})
		if err != nil {
			logger.Error("启动环境容器错误！", err.Error())
			os.Exit(1)
		}
		logger.Info("环境容器启动成功！")
	} else {
		var err error
		SettingDB, err = dingnuts.NewDingNuts(filepath.Join(RunPath, "setting"))
		if err != nil {
			logger.Error("打开设置数据库失败！", err.Error())
			os.Exit(1)
		}
		ContainerID = tools.GetValue(SettingDB, logger, "containerID")
		*ContainerPort = tools.GetValue(SettingDB, logger, "containerPort")
		logger.Debug(ContainerID, *ContainerPort)
		if !tools.CheckContainer(DockerClient, context.Background(), logger, ContainerID) {
			logger.Error("环境容器不存在或image不存在！")
			logger.Error("请使用clean选项清理容器和image，重新初始化")
			os.Exit(1)
		}
		tools.StartOrRestartConstainer(DockerClient, context.Background(), logger, ContainerID)

	}

	if !tools.PathExists(filepath.Join(RunPath, "setting")) {
		var err error
		SettingDB, err = dingnuts.NewDingNuts(filepath.Join(RunPath, "setting"))
		if err != nil {
			logger.Error("打开设置数据库失败！", err.Error())
			os.Exit(1)
		}
		tools.SetValue(SettingDB, logger, "containerID", ContainerID)
		tools.SetValue(SettingDB, logger, "containerPort", *ContainerPort)
	}

	tools.StartExec(context.Background(), DockerClient, logger, ContainerID, *ContainerPort)

	logger.Info("正在启动本地控制服务器端...")

	mainServer := gin.Default()

	api_group := mainServer.Group("/api")

	api_group.POST("/check", func(ctx *gin.Context) {
		res := tools.CheckEvery(context.Background(), DockerClient, logger, ContainerID)
		if res == 0 {
			ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
			return
		}
		if res == 1 {
			ctx.JSON(http.StatusOK, gin.H{"status": "error", "message": "container not running..."})
			return
		}
		if res == 2 {
			ctx.JSON(http.StatusOK, gin.H{"status": "error", "message": "exec not running..."})
			return
		}
	})

	api_group.POST("/getapi", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok", "api": "http://127.0.0.1:" + *ContainerPort + "/run"})
	})

	api_group.POST("/fix", func(ctx *gin.Context) {
		stop_time := time.Duration(2 * time.Second)
		err := DockerClient.ContainerStop(ctx, ContainerID, &stop_time)
		if err != nil {
			ctx.JSON(http.StatusOK, gin.H{"status": "error", "message": "容器stop失败", "error": err.Error()})
			return
		}
		err = DockerClient.ContainerRemove(context.Background(), ContainerID, types.ContainerRemoveOptions{})
		if err != nil {
			ctx.JSON(http.StatusOK, gin.H{"status": "error", "message": "容器移除失败", "error": err.Error()})
			return
		}
		port, _ := nat.NewPort("tcp", *ContainerPort)
		res_create, err := DockerClient.ContainerCreate(context.Background(), &container.Config{
			Image:        "codedocker",
			Tty:          true,
			ExposedPorts: nat.PortSet{port: struct{}{}},
		}, &container.HostConfig{
			PortBindings: nat.PortMap{port: []nat.PortBinding{
				{HostIP: "0.0.0.0", HostPort: string(port)},
			}},
		}, nil, nil, "codedockerC")
		if err != nil {
			ctx.JSON(http.StatusOK, gin.H{"status": "error", "message": "容器创建失败", "error": err.Error()})
			return
		}
		ContainerID = res_create.ID
		err = DockerClient.ContainerStart(ctx, ContainerID, types.ContainerStartOptions{})
		if err != nil {
			ctx.JSON(http.StatusOK, gin.H{"status": "error", "message": "容器启动失败", "error": err.Error()})
			return
		}
		tools.SetValue(SettingDB, logger, "containerID", ContainerID)
		tools.StartExec(context.Background(), DockerClient, logger, ContainerID, *ContainerPort)
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	err := mainServer.Run(":" + *Port)
	if err != nil {
		logger.Error("启动本地控制端服务器失败！", err.Error())
	}
}
