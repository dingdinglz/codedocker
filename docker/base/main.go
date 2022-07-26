package main

import (
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/dingdinglz/dingtools/dinglog"
	"github.com/gin-contrib/timeout"
	"github.com/gin-gonic/gin"
)

var (
	logger      *dinglog.DingLogger
	Debug       *bool
	MainServer  *gin.Engine
	Port        *string
	DoneNums    int = 0
	PhpDoneNums int = 0
	RunPath     string
)

func init() {
	Debug = flag.Bool("debug", false, "Debug Mode")
	Port = flag.String("port", "7000", "启动端口")
	flag.Parse()
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func RunCmd(path string, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = path
	cmd.Run()
}

func RunCmdAndIsOkAndGetOut(path string, name string, args ...string) (bool, error, string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = path
	res, err := cmd.CombinedOutput()
	if err != nil {
		return false, err, ""
	}
	return true, nil, string(res)
}

func build(ctx *gin.Context) {
	code := ctx.PostForm("code")
	language := ctx.PostForm("language")
	if language == "go" {
		if !PathExists(filepath.Join(RunPath, "go_cache")) {
			err := os.Mkdir(filepath.Join(RunPath, "go_cache"), 0777)
			if err != nil {
				ctx.JSON(http.StatusOK, gin.H{"status": 1, "message": err.Error()})
				return
			}
		}
		folder_name := strconv.Itoa(DoneNums + 1)
		DoneNums++
		err := os.Mkdir(filepath.Join(RunPath, "go_cache", folder_name), 0777)
		if err != nil {
			ctx.JSON(http.StatusOK, gin.H{"status": 1, "message": err.Error()})
			return
		}
		err = ioutil.WriteFile(filepath.Join(RunPath, "go_cache", folder_name, "main.go"), []byte(code), 0777)
		if err != nil {
			ctx.JSON(http.StatusOK, gin.H{"status": 2, "message": err.Error()})
			return
		}
		folder_path := filepath.Join(RunPath, "go_cache", folder_name)
		RunCmd(folder_path, "/go/bin/go", "mod", "init", "out")
		RunCmd(folder_path, "/go/bin/go", "mod", "tidy")
		result, _, outstring := RunCmdAndIsOkAndGetOut(folder_path, "/go/bin/go", "build")
		if !result {
			ctx.JSON(http.StatusOK, gin.H{"status": 3, "message": outstring})
			return
		}
		result, err_build, outstring := RunCmdAndIsOkAndGetOut(folder_path, filepath.Join(folder_path, "out"))
		if !result {
			ctx.JSON(http.StatusOK, gin.H{"status": 4, "message": err_build.Error()})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"status": 0, "message": outstring})
		return
	}
	if language == "php" {
		if !PathExists(filepath.Join(RunPath, "php_cache")) {
			err := os.Mkdir(filepath.Join(RunPath, "php_cache"), 0777)
			if err != nil {
				ctx.JSON(http.StatusOK, gin.H{"status": 1, "message": err.Error()})
				return
			}
		}
		file_name := strconv.Itoa(PhpDoneNums+1) + ".php"
		PhpDoneNums++
		err := ioutil.WriteFile(filepath.Join(RunPath, "php_cache", file_name), []byte(code), 0777)
		if err != nil {
			ctx.JSON(http.StatusOK, gin.H{"status": 2, "message": err.Error()})
			return
		}
		_, err, out_string := RunCmdAndIsOkAndGetOut(filepath.Join(RunPath, "php_cache"), "php", filepath.Join(RunPath, "php_cache", file_name))
		if err != nil {
			ctx.JSON(http.StatusOK, gin.H{"status": 3, "message": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"status": 0, "message": out_string})
		return
	}
}

func main() {
	RunPath, _ = os.Getwd()
	logger = dinglog.NewLogger()
	if *Debug {
		logger.SetLevel(dinglog.Level_Debug)
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	logger.Info("正在清理build cache...")

	err := os.RemoveAll(filepath.Join(RunPath, "go_cache"))
	if err != nil {
		logger.Error("清理build cache错误", err.Error())
		os.Exit(1)
	}

	err = os.RemoveAll(filepath.Join(RunPath, "php_cache"))
	if err != nil {
		logger.Error("清理build cache错误", err.Error())
		os.Exit(1)
	}

	MainServer := gin.Default()
	logger.Info("正在绑定路由...")

	MainServer.POST("/run", timeout.New(
		timeout.WithTimeout(30*time.Second),
		timeout.WithHandler(build),
		timeout.WithResponse(func(ctx *gin.Context) {
			ctx.JSON(http.StatusOK, gin.H{"status": 1000, "message": "time out!"})
		}),
	))

	logger.Info("正在启动后端...")
	err = MainServer.Run(":" + *Port)
	if err != nil {
		logger.Error("启动后端失败！", err.Error())
		return
	}
}
