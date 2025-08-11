package util

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
	"go.uber.org/zap"
	"io"
	"logo_api/settings"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

type CosClient struct {
	Client *cos.Client
}
type BitmapResourceInfo struct {
	ShortName        string
	ResourceName     string
	ResourceMd5      string
	ResourceSizeB    int64
	ResolutionWidth  int64
	ResolutionHeight int64
	BackgroundColor  string
}

// NewClient 创建新*CosClient 对象，以便执行各个方法
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

// GetObjectByResourceName 直接从腾讯云COS上获取资源
func (c *CosClient) GetObjectByResourceName(resourceName string, shortName string) (data []byte, err error) {
	name := fmt.Sprintf("beacon/%s/%s", shortName, resourceName)
	// 直接用 SDK 的 Get 方法拿到 io.ReadCloser
	resp, err := c.Client.Object.Get(context.Background(), name, nil)
	if err != nil {
		zap.L().Error("cos.Object.Get() err:", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	data, err = io.ReadAll(resp.Body)
	if err != nil {
		zap.L().Error("io.ReadAll() err:", zap.Error(err))
		return nil, err
	}

	return data, nil
}

// GetObjectByResourceNameAndSvgToBitmap 从腾讯云COS上获取矢量图资源，并进行格式转换，最后返回位图相关信息
func (c *CosClient) GetObjectByResourceNameAndSvgToBitmap(resourceName, title, shortName, resourceType string, size int, width int, height int, bgColor string) (data []byte, bitmapInfo BitmapResourceInfo, err error) {
	// 创建临时文件（系统临时目录下，自动生成唯一文件名）
	tmpFile, err := os.CreateTemp("", resourceName) // "" 表示系统临时目录
	if err != nil {
		zap.L().Error("os.CreateTemp() err:", zap.Error(err))
		return nil, BitmapResourceInfo{}, err
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	cosPath := fmt.Sprintf("beacon/%s/%s", shortName, resourceName)
	resp, err := c.Client.Object.Get(context.Background(), cosPath, nil)
	if err != nil {
		zap.L().Error("cos.Object.Get() err:", zap.Error(err))
		return nil, BitmapResourceInfo{}, err
	}
	defer resp.Body.Close()

	// 直接把远程数据写入临时文件
	if _, err = io.Copy(tmpFile, resp.Body); err != nil {
		zap.L().Error("io.Copy() err:", zap.Error(err))
		return nil, BitmapResourceInfo{}, err
	}
	// 关闭临时文件用于后续读取
	if err = tmpFile.Close(); err != nil {
		zap.L().Error("tmpFile.Close() err:", zap.Error(err))
		return nil, BitmapResourceInfo{}, err
	}
	// 临时 svg 文件路径
	svgPath := tmpFile.Name()

	// 例如转换生成临时 bitmap 文件
	bitmapTmpFile, err := os.CreateTemp("", fmt.Sprintf("%s-logo-*.%s", title, resourceType))
	if err != nil {
		zap.L().Error("os.CreateTemp() err:", zap.Error(err))
		return nil, BitmapResourceInfo{}, err
	}
	defer func() {
		bitmapTmpFile.Close()
		os.Remove(bitmapTmpFile.Name())
	}()

	bitmapPath := bitmapTmpFile.Name()
	bitmapTmpFile.Close() // 关闭后传路径给转换命令使用

	// 调用 rsvg-convert 执行格式转换
	if err = ConvertSvgToBitmap(svgPath, bitmapPath, resourceType, size, width, height, bgColor); err != nil {
		zap.L().Error("ConvertSvgToBitmap() err:", zap.Error(err))
		return nil, BitmapResourceInfo{}, err
	}

	// 读取 bitmap 文件的 md5 和 size
	fileMd5, err := GetFileMd5(bitmapPath)
	if err != nil {
		zap.L().Error("GetFileMd5() err:", zap.Error(err))
		return nil, BitmapResourceInfo{}, err
	}
	sizeb, err := GetFileSizeb(bitmapPath)
	if err != nil {
		zap.L().Error("GetFileSize() err:", zap.Error(err))
		return nil, BitmapResourceInfo{}, err
	}

	// 读转换后的图片文件内容，准备返回
	data, err = os.ReadFile(bitmapPath)
	if err != nil {
		zap.L().Error("os.ReadFile() err:", zap.Error(err))
		return nil, BitmapResourceInfo{}, err
	}

	var newFileName, resBgColor string
	var resWidth, resHeight int64
	if size > 0 {
		if bgColor == "" {
			newFileName = fmt.Sprintf("%s-logo-%dpx.%s", title, size, resourceType)
		} else {
			newFileName = fmt.Sprintf("%s-logo-%dpx-%s.%s", title, size, bgColor, resourceType)
		}
		resWidth, resHeight = int64(size), int64(size)
	} else if width > 0 && height > 0 {
		if bgColor == "" {
			newFileName = fmt.Sprintf("%s-logo-%dpx-%dpx.%s", title, width, height, resourceType)
		} else {
			newFileName = fmt.Sprintf("%s-logo-%dpx-%dpx-%s.%s", title, width, height, bgColor, resourceType)
		}
		resWidth, resHeight = int64(width), int64(height)
	}
	if bgColor != "" {
		resBgColor = bgColor
	} else {
		resBgColor = "#FFFFFF"
	}
	// 上传到腾讯云 COS
	uploadCosPath := fmt.Sprintf("beacon/%s/%s", shortName, newFileName)
	if err = c.UploadObject(bitmapPath, uploadCosPath); err != nil {
		zap.L().Error("UploadObject() err:", zap.Error(err))
		return nil, BitmapResourceInfo{}, err
	}

	// 返回信息给 service 层
	info := BitmapResourceInfo{
		ShortName:        shortName,
		ResourceName:     newFileName,
		ResourceMd5:      fileMd5,
		ResourceSizeB:    sizeb,
		ResolutionWidth:  resWidth,
		ResolutionHeight: resHeight,
		BackgroundColor:  resBgColor,
	}
	return data, info, nil
}

// ConvertSvgToBitmap 使用 rsvg-convert 命令行工具，对临时下载的 svg 文件进行转格式操作
func ConvertSvgToBitmap(svgPath, bitmapPath, resourceType string, size, width, height int, bgColor string) error {
	// 校验传入格式合法性
	validFormats := map[string]bool{
		"png":  true,
		"jpg":  true,
		"jpeg": true,
		"webp": true,
	}
	if !validFormats[resourceType] {
		zap.L().Error("resourceType is not valid",
			zap.String("resourceType", resourceType))
		return fmt.Errorf("unsupported format: %s", resourceType)
	}
	svgPathWsl := windowsPathToWslPath(svgPath)
	bitmapPathWsl := windowsPathToWslPath(bitmapPath)

	// 构造参数
	args := []string{"-f", resourceType, "-o", bitmapPathWsl, svgPathWsl}

	if size > 0 {
		args = append([]string{"-w", fmt.Sprint(size), "-h", fmt.Sprint(size)}, args...)
	} else {
		if width > 0 {
			args = append([]string{"-w", fmt.Sprint(width)}, args...)
		}
		if height > 0 {
			args = append([]string{"-h", fmt.Sprint(height)}, args...)
		}
	}

	if bgColor != "" {
		args = append(args, "--background-color="+bgColor) // 注意这里，把背景参数加到最后
	}
	zap.L().Debug("Running rsvg-convert", zap.Strings("args", args))

	// 调用 wsl 运行 rsvg-convert
	cmd := exec.Command("wsl", append([]string{"rsvg-convert"}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		zap.L().Error("cmd.Run() err:", zap.Error(err))
		return fmt.Errorf("convert failed: %v, output: %s", err, string(output))
	}

	return nil
}

// GetFileSizeb 获取文件Size(以b为单位)
func GetFileSizeb(filepath string) (int64, error) {
	fileInfo, err := os.Stat(filepath)
	if err != nil {
		zap.L().Error("os.Stat() err:", zap.Error(err))
		return 0, err
	}
	return fileInfo.Size(), nil
}

// GetFileMd5 获取文件Md5值（十六进制字符串）
func GetFileMd5(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		zap.L().Error("os.Open() err:", zap.Error(err))
		return "", err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		zap.L().Error("io.Copy() err:", zap.Error(err))
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// UploadObject 根据 本地路径 和 腾讯云cos路径，上传文件
func (c *CosClient) UploadObject(localPath, cosPath string) error {
	// 打开本地文件
	file, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 上传文件到 COS（覆盖同名文件）
	_, err = c.Client.Object.Put(context.Background(), cosPath, file, nil)
	return err
}

// windowsPathToWslPath 将 Windows 路径转换成 WSL 下的 Linux 路径格式
func windowsPathToWslPath(winPath string) string {
	if len(winPath) < 3 {
		return winPath // 非标准路径，原样返回
	}

	// 取盘符小写，比如 C -> c
	driveLetter := strings.ToLower(string(winPath[0]))

	// 去掉盘符和冒号（例如 C:），并把反斜杠替换成斜杠
	pathPart := strings.ReplaceAll(winPath[2:], "\\", "/")

	// 组合成 WSL 路径格式
	wslPath := fmt.Sprintf("/mnt/%s/%s", driveLetter, pathPart)

	return wslPath
}

// DeleteObject 删除腾讯云COS中对应路径的资源
func (c *CosClient) DeleteObject(path string) error {
	_, err := c.Client.Object.Delete(context.Background(), path)
	if err != nil {
		zap.L().Error("DeleteObject() err:", zap.String("path", path), zap.Error(err))
		return err
	}
	zap.L().Info("DeleteObject() success", zap.String("path", path))
	return nil
}
