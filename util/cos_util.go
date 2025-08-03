package util

import (
	"context"
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
	"go.uber.org/zap"
	"io"
	"logo_api/settings"
	"net/http"
	"net/url"
	"os"
)

type CosClient struct {
	Client *cos.Client
}

func NewClient(config *settings.CosConfig) (*CosClient, error) {
	u, err := url.Parse(config.BucketUrl)
	if err != nil {
		zap.L().Error("url.Parse() err:", zap.Error(err))
	}
	b := &cos.BaseURL{BucketURL: u}
	// 初始化客户端
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  config.SecretId,
			SecretKey: config.SecretKey,
		},
	})
	cosClient := &CosClient{Client: client}
	return cosClient, err
}

func (c *CosClient) UploadObject(shortName string) (err error) {
	name := fmt.Sprintf("beacon/%s/%s_logo.svg", shortName, shortName) // svg? ???

	// 1 直接用 SDK 的 Get 方法拿到 io.ReadCloser
	resp, err := c.Client.Object.Get(context.Background(), name, nil)
	if err != nil {
		zap.L().Error("cos.Object.Get() err:", zap.Error(err))
		return err
	}

	// 2 创建目录，如果不存在
	err = os.MkdirAll(shortName, os.ModePerm)
	if err != nil {
		zap.L().Error("os.MkdirAll() err:", zap.Error(err))
		return err
	}

	// 3 在本地创建一个文件
	filePath := fmt.Sprintf("%s/%s_logo.svg", shortName, shortName)
	f, err := os.Create(filePath)
	if err != nil {
		zap.L().Error("os.Create() err:", zap.Error(err))
		return err
	}
	defer f.Close()

	// 4 把远程的数据流拷贝到本地文件
	_, err = io.Copy(f, resp.Body)
	if err != nil {
		zap.L().Error("io.Copy() err:", zap.Error(err))
		return err
	}

	zap.L().Info("成功下载文件到：" + shortName)
	return nil
}
