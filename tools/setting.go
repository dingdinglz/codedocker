package tools

import (
	"os"

	"github.com/dingdinglz/dingtools/dingdb/dingnuts"
	"github.com/dingdinglz/dingtools/dinglog"
)

func SetValue(DB *dingnuts.DingNuts, logger *dinglog.DingLogger, name string, value string) {
	err := DB.SetValue([]byte(name), []byte(value))
	if err != nil {
		logger.Error("数据库存放错误！", err.Error())
		os.Exit(1)
	}
}

func GetValue(DB *dingnuts.DingNuts, logger *dinglog.DingLogger, name string) string {
	res, err := DB.GetValue([]byte(name))
	if err != nil {
		logger.Error("数据库读数据错误！", err.Error())
		os.Exit(1)
	}
	return string(res)
}
