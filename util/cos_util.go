package util

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
	"go.uber.org/zap"
	"io"
	"logo_api/dao/mysql"
	"logo_api/settings"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
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

func (c *CosClient) DownloadObjectByResourceName(resourceName string, shortName string) (err error) {
	name := fmt.Sprintf("beacon/%s/%s", shortName, resourceName)
	// 1 直接用 SDK 的 Get 方法拿到 io.ReadCloser
	resp, err := c.Client.Object.Get(context.Background(), name, nil)
	if err != nil {
		zap.L().Error("cos.Object.Get() err:", zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	// 2 创建目录，如果不存在
	err = os.MkdirAll(shortName, os.ModePerm)
	if err != nil {
		zap.L().Error("os.MkdirAll() err:", zap.Error(err))
		return err
	}

	// 3 在本地创建一个文件
	filePath := filepath.Join(shortName, resourceName)
	f, err := os.Create(filePath)
	if err != nil {
		zap.L().Error("os.Create() err:", zap.Error(err))
		return err
	}
	defer f.Close()

	// 4 把远程的数据流拷贝到本地文件
	n, err := io.Copy(f, resp.Body)
	if err != nil {
		zap.L().Error("io.Copy() err:", zap.Error(err))
		return err
	}

	zap.L().Info("成功下载文件到本地", zap.String("filePath", filePath), zap.Int64("bytes", n))
	// 确认文件大小
	fi, err := os.Stat(filePath)
	if err != nil {
		zap.L().Error("os.Stat() err:", zap.Error(err))
		return err
	}
	zap.L().Info("文件大小", zap.Int64("size", fi.Size()))

	return nil
}

func (c *CosClient) DownloadObjectByResourceNameAndSvgToBitmap(resourceName, title, shortName, resourceType string, size int, width int, height int, bgColor string) (err error) {

	// 获取跨平台临时目录
	tmpDir := os.TempDir()

	svgPath := filepath.Join(tmpDir, resourceName)

	// 确保父目录存在
	if err = os.MkdirAll(filepath.Dir(svgPath), os.ModePerm); err != nil {
		zap.L().Error("os.MkdirAll() err:", zap.Error(err))
		return err
	}

	// 创建文件
	outFile, err := os.Create(svgPath)
	if err != nil {
		zap.L().Error("os.Create() err:", zap.Error(err))
		return err
	}
	defer outFile.Close()

	name := fmt.Sprintf("beacon/%s/%s", shortName, resourceName)
	// 1 直接用 SDK 的 Get 方法拿到 io.ReadCloser
	resp, err := c.Client.Object.Get(context.Background(), name, nil)
	if err != nil {
		zap.L().Error("cos.Object.Get() err:", zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	// 把 svg 文件从 COS 下载到本地
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		zap.L().Error("io.Copy() err:", zap.Error(err))
		return err
	}

	// 调用 rsvg-convert 执行格式转换
	var bitmapPath string // 输出路径
	if size != 0 {
		bitmapPath = filepath.Join(tmpDir, fmt.Sprintf("%s-logo-%dpx.%s", title, size, resourceType))
	} else {
		if width != 0 && height != 0 {
			bitmapPath = filepath.Join(tmpDir, fmt.Sprintf("%s-logo-%dpx-%dpx.%s", title, width, height, resourceType))
		}
	}
	if err = os.MkdirAll(filepath.Dir(bitmapPath), os.ModePerm); err != nil {
		zap.L().Error("os.MkdirAll(bitmapPath) err:", zap.Error(err))
		return err
	}

	if err = ConvertSvgToBitmap(svgPath, bitmapPath, resourceType, size, width, height, bgColor); err != nil {
		zap.L().Error("ConvertSvgToBitmap() err:", zap.Error(err))
		return err
	}

	// 3. 转换完成后，确认文件存在
	if _, err := os.Stat(bitmapPath); os.IsNotExist(err) {
		zap.L().Error("bitmap file missing after conversion", zap.String("path", bitmapPath))
		return fmt.Errorf("bitmap file not found after conversion")
	}

	localPath, err := SaveFileToLocalDir(bitmapPath, shortName)
	if err != nil {
		return err
	}
	zap.L().Info("文件保存到本地目录", zap.String("localPath", localPath))

	// 同步在 腾讯云 cos 和 本地 MySQL 中同步存储新文件，并把新文件保存到本地。
	outputDir := filepath.Join(".", shortName)
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return err
	}

	var bitmapResource settings.UniversityResources
	if size != 0 {
		bitmapResource.ResourceName = fmt.Sprintf("%s-logo-%dpx.%s", title, size, resourceType)
		bitmapResource.ResolutionHeight = sql.NullInt64{
			Int64: int64(size),
			Valid: true}
		bitmapResource.ResolutionWidth = sql.NullInt64{
			Int64: int64(size),
			Valid: true,
		}
	} else {
		if width != 0 && height != 0 {
			bitmapResource.ResourceName = fmt.Sprintf("%s-logo-%dpx-%dpx.%s", title, width, height, resourceType)
			bitmapResource.ResolutionWidth = sql.NullInt64{
				Int64: int64(width),
				Valid: true,
			}
			bitmapResource.ResolutionHeight = sql.NullInt64{
				Int64: int64(height),
				Valid: true,
			}
		}
	}
	bitmapResource.ShortName = shortName
	bitmapResource.Title = title
	bitmapResource.ResourceType = resourceType
	fileMd5, err := GetFileMd5(bitmapPath)
	if err != nil {
		zap.L().Error("GetFileMd5() err:", zap.Error(err))
		return err
	}
	bitmapResource.ResourceMd5 = fileMd5

	sizeb, err := GetFileSizeb(bitmapPath)
	if err != nil {
		zap.L().Error("GetFileSizeb() err:", zap.Error(err))
		return err
	}
	bitmapResource.ResourceSizeB = sql.NullInt64{
		Int64: sizeb,
		Valid: true,
	}
	now := time.Now()
	bitmapResource.LastUpdateTime = sql.NullTime{
		Time:  now,
		Valid: true,
	} // sql.NullTime
	bitmapResource.IsVector = false
	bitmapResource.IsBitmap = true
	bitmapResource.UsedForEdge = false
	bitmapResource.IsDeleted = false

	if err = mysql.InsertUniversityResource(bitmapResource); err != nil {
		zap.L().Error("mysql.InsertUniversityResource() err:", zap.Error(err))
		return err
	}

	// 上传到腾讯云 COS
	uploadCosPath := fmt.Sprintf("beacon/%s/%s", shortName, bitmapResource.ResourceName)
	if err = c.UploadObject(bitmapPath, uploadCosPath); err != nil {
		zap.L().Error("UploadOnject() err:", zap.Error(err))
		return err
	}
	return nil
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
		args = append([]string{"--background-color=" + bgColor}, args...)
	}

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

// SaveFileToLocalDir 将源文件复制到目标目录，目标文件名和源文件相同
func SaveFileToLocalDir(srcFilePath, targetDir string) (string, error) {
	// 创建目标目录（如果不存在）
	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		zap.L().Error("os.MkdirAll() err:", zap.Error(err))
		return "", err
	}

	// 目标文件路径
	targetFilePath := filepath.Join(targetDir, filepath.Base(srcFilePath))

	// 打开源文件
	srcFile, err := os.Open(srcFilePath)
	if err != nil {
		zap.L().Error("os.Open() err:", zap.Error(err))
		return "", err
	}
	defer srcFile.Close()

	// 创建目标文件
	dstFile, err := os.Create(targetFilePath)
	if err != nil {
		zap.L().Error("os.Create() err:", zap.Error(err))
		return "", err
	}
	defer dstFile.Close()

	// 拷贝文件内容
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		zap.L().Error("io.Copy() err:", zap.Error(err))
		return "", err
	}

	zap.L().Info("成功保存文件到本地目录", zap.String("path", targetFilePath))
	return targetFilePath, nil
}
