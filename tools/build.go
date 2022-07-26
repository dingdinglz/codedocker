package tools

import (
	"archive/tar"
	"bytes"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/dingdinglz/dingtools/dinglog"
)

func BuildDockerFileTar(logger *dinglog.DingLogger, rootPath string) {
	if PathExists(filepath.Join(rootPath, "docker", "Dockerfile.tar.gz")) {
		logger.Info("正在清理Dockerfile缓存...")
		os.Remove(filepath.Join(rootPath, "docker", "Dockerfile.tar.gz"))
	}
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	tarRootPath := filepath.Join(rootPath, "docker")
	filepath.Walk(tarRootPath, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		tarPath := strings.ReplaceAll(path, tarRootPath, "")
		tarPath = tarPath[1:]
		logger.Debug(tarPath)
		file, err_open := os.OpenFile(path, os.O_RDONLY, 0777)
		if err_open != nil {
			logger.Error("打开文件", tarPath, "失败！", err_open.Error())
			os.Exit(1)
			return nil
		}
		filesize, _ := file.Stat()
		header := &tar.Header{
			Name: tarPath,
			Size: filesize.Size(),
			Mode: 0777,
		}
		err_w_header := tw.WriteHeader(header)
		if err_w_header != nil {
			logger.Error("写tar头信息错误", tarPath, err_w_header.Error())
			os.Exit(1)
			return nil
		}
		file_text, _ := ioutil.ReadAll(file)
		_, err_w_write := tw.Write(file_text)
		if err_w_write != nil {
			logger.Error("写tar错误", tarPath, err_w_write.Error())
			os.Exit(1)
			return nil
		}
		return nil
	})
	tw.Close()
	wf, err := os.OpenFile(filepath.Join(rootPath, "docker", "Dockerfile.tar.gz"), os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		logger.Error("打开tar文件错误", err.Error())
		os.Exit(1)
	}
	buf.WriteTo(wf)
}
