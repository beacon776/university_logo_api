package util

import (
	"crypto/md5"
	"encoding/hex"
	"image"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
)

func CalculateMD5(fileHeader *multipart.FileHeader) (string, error) {
	f, _ := fileHeader.Open()
	defer f.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func GetImageInfo(fileHeader *multipart.FileHeader) (width, height int, isVector, isBitmap int) {
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))

	// 1. 判断是否为矢量图 (以 SVG 为例)
	if ext == ".svg" {
		return 0, 0, 1, 0 // SVG 通常不通过 image.Decode 获取固定宽高
	}

	// 2. 判断是否为位图并获取宽高
	f, _ := fileHeader.Open()
	defer f.Close()

	img, _, err := image.DecodeConfig(f) // 只读取配置，不加载全图，效率高
	if err == nil {
		return img.Width, img.Height, 0, 1
	}

	return 0, 0, 0, 0
}
