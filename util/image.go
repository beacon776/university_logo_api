package util

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	xdraw "golang.org/x/image/draw"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// W3C 标准颜色名称映射 (140个常用颜色)
var colorNames = map[string]string{
	"aliceblue": "F0F8FF", "antiquewhite": "FAEBD7", "aqua": "00FFFF", "aquamarine": "7FFFD4", "azure": "F0FFFF",
	"beige": "F5F5DC", "bisque": "FFE4C4", "black": "000000", "blanchedalmond": "FFEBCD", "blue": "0000FF",
	"blueviolet": "8A2BE2", "brown": "A52A2A", "burlywood": "DEB887", "cadetblue": "5F9EA0", "chartreuse": "7FFF00",
	"chocolate": "D2691E", "coral": "FF7F50", "cornflowerblue": "6495ED", "cornsilk": "FFF8DC", "crimson": "DC143C",
	"cyan": "00FFFF", "darkblue": "00008B", "darkcyan": "008B8B", "darkgoldenrod": "B8860B", "darkgray": "A9A9A9",
	"darkgreen": "006400", "darkkhaki": "BDB76B", "darkmagenta": "8B008B", "darkolivegreen": "556B2F", "darkorange": "FF8C00",
	"darkorchid": "9932CC", "darkred": "8B0000", "darksalmon": "E9967A", "darkseagreen": "8FBC8F", "darkslateblue": "483D8B",
	"darkslategray": "2F4F4F", "darkturquoise": "00CED1", "darkviolet": "9400D3", "deeppink": "FF1493", "deepskyblue": "00BFFF",
	"dimgray": "696969", "dodgerblue": "1E90FF", "firebrick": "B22222", "floralwhite": "FFFAF0", "forestgreen": "228B22",
	"fuchsia": "FF00FF", "gainsboro": "DCDCDC", "ghostwhite": "F8F8FF", "gold": "FFD700", "goldenrod": "DAA520",
	"gray": "808080", "green": "008000", "greenyellow": "ADFF2F", "honeydew": "F0FFF0", "hotpink": "FF69B4",
	"indianred": "CD5C5C", "indigo": "4B0082", "ivory": "FFFFF0", "khaki": "F0E68C", "lavender": "E6E6FA",
	"lavenderblush": "FFF0F5", "lawngreen": "7CFC00", "lemonchiffon": "FFFACD", "lightblue": "ADD8E6", "lightcoral": "F08080",
	"lightcyan": "E0FFFF", "lightgoldenrodyellow": "FAFAD2", "lightgray": "D3D3D3", "lightgreen": "90EE90", "lightpink": "FFB6C1",
	"lightsalmon": "FFA07A", "lightseagreen": "20B2AA", "lightskyblue": "87CEFA", "lightslategray": "778899", "lightsteelblue": "B0C4DE",
	"lightyellow": "FFFFE0", "lime": "00FF00", "limegreen": "32CD32", "linen": "FAF0E6", "magenta": "FF00FF",
	"maroon": "800000", "mediumaquamarine": "66CDAA", "mediumblue": "0000CD", "mediumorchid": "BA55D3", "mediumpurple": "9370DB",
	"mediumseagreen": "3CB371", "mediumslateblue": "7B68EE", "mediumspringgreen": "00FA9A", "mediumturquoise": "48D1CC", "mediumvioletred": "C71585",
	"midnightblue": "191970", "mintcream": "F5FFFA", "mistyrose": "FFE4E1", "moccasin": "FFE4B5", "navajowhite": "FFDEAD",
	"navy": "000080", "oldlace": "FDF5E6", "olive": "808000", "olivedrab": "6B8E23", "orange": "FFA500",
	"orangered": "FF4500", "orchid": "DA70D6", "palegoldenrod": "EEE8AA", "palegreen": "98FB98", "paleturquoise": "AFEEEE",
	"palevioletred": "DB7093", "papayawhip": "FFEFD5", "peachpuff": "FFDAB9", "peru": "CD853F", "pink": "FFC0CB",
	"plum": "DDA0DD", "powderblue": "B0E0E6", "purple": "800080", "rebeccapurple": "663399", "red": "FF0000",
	"rosybrown": "BC8F8F", "royalblue": "4169E1", "saddlebrown": "8B4513", "salmon": "FA8072", "sandybrown": "F4A460",
	"seagreen": "2E8B57", "seashell": "FFF5EE", "sienna": "A0522D", "silver": "C0C0C0", "skyblue": "87CEEB",
	"slateblue": "6A5ACD", "slategray": "708090", "snow": "FFFAFA", "springgreen": "00FF7F", "steelblue": "4682B4",
	"tan": "D2B48C", "teal": "008080", "thistle": "D8BFD8", "tomato": "FF6347", "turquoise": "40E0D0",
	"violet": "EE82EE", "wheat": "F5DEB3", "white": "FFFFFF", "whitesmoke": "F5F5F5", "yellow": "FFFF00",
	"yellowgreen": "9ACD32",
}

// 这里的正则支持 rgb(r,g,b) 和 rgba(r,g,b,a)
// 它允许有逗号或空格分隔，适配性更强
var colorRegex = regexp.MustCompile(`rgba?\(\s*(\d{1,3})\s*[\s,]\s*(\d{1,3})\s*[\s,]\s*(\d{1,3})(?:\s*[\s,]\s*[\d\.]+)?\s*\)`)

// NormalizeColor 统一将各种颜色格式转换为 #RRGGBB 格式
func NormalizeColor(bgColor string) string {
	clean := strings.ToLower(strings.TrimSpace(bgColor))
	if clean == "" || clean == "transparent" {
		return ""
	}

	// 1. 处理颜色名称
	if hex, ok := colorNames[clean]; ok {
		return "#" + hex
	}

	// 2. 处理 RGB / RGBA
	// 注意：RGBA 的 A (透明度) 在你的业务场景中通常被丢弃，只取 RGB
	matches := colorRegex.FindStringSubmatch(clean)
	if len(matches) >= 4 {
		r, _ := strconv.Atoi(matches[1])
		g, _ := strconv.Atoi(matches[2])
		b, _ := strconv.Atoi(matches[3])
		if r <= 255 && g <= 255 && b <= 255 {
			return fmt.Sprintf("#%02X%02X%02X", r, g, b)
		}
	}

	// 3. 处理 HEX
	hex := strings.ToUpper(strings.TrimPrefix(clean, "#"))

	switch len(hex) {
	case 3: // RGB -> RRGGBB
		return "#" + string(hex[0]) + string(hex[0]) + string(hex[1]) + string(hex[1]) + string(hex[2]) + string(hex[2])
	case 4: // ARGB -> RRGGBB (丢弃 A)
		return "#" + string(hex[0]) + string(hex[0]) + string(hex[1]) + string(hex[1]) + string(hex[2]) + string(hex[2])
	case 6: // RRGGBB
		return "#" + hex
	case 8: // RRGGBBAA -> RRGGBB (丢弃 AA)
		return "#" + hex[0:6]
	}

	return ""
}

// 把任意 image.Image 叠到统一底色
func ImageNewRGBAWithBG(src image.Image, bg color.Color) *image.RGBA {
	dst := image.NewRGBA(src.Bounds())
	draw.Draw(dst, dst.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), src, src.Bounds().Min, draw.Over)
	return dst
}

// 解析 "#RRGGBB"/"#RGB"，失败则返回白色
func ParseHexOrWhite(s string) color.RGBA {
	if s == "" {
		return color.RGBA{255, 255, 255, 255}
	}
	var r, g, b uint8
	if len(s) == 7 && s[0] == '#' {
		_, err := fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b)
		if err == nil {
			return color.RGBA{r, g, b, 255}
		}
	}
	if len(s) == 4 && s[0] == '#' {
		var r1, g1, b1 uint8
		if _, err := fmt.Sscanf(s, "#%1x%1x%1x", &r1, &g1, &b1); err == nil {
			return color.RGBA{r1 * 17, g1 * 17, b1 * 17, 255}
		}
	}
	return color.RGBA{255, 255, 255, 255}
}

// ConvertSvgToBitmap 使用 rsvg-convert 命令行工具，对临时下载的 svg 文件进行转格式操作
// 注意：rsvg-convert 只支持 svg 转 png，如果是其他格式的话，需要再调用 ConvertPngToOther
func ConvertSvgToBitmap(svgPath, bitmapPath, resourceType string, size, width, height int, bgColor string) error {
	runMode := viper.GetString("RUN_MODE") // 引入 runMode 变量
	// 第一步：先进行 svg 转 png，格式校验放在 ConvertPngToOther 里
	targetSize := size
	if targetSize == 0 && width > 0 && height > 0 {
		targetSize = min(width, height)
	}
	if targetSize == 0 {
		zap.L().Error("targetSize is zero")
		return nil
	}
	var cmd *exec.Cmd
	if runMode == "local" {

		// Win 调用 WSL 运行命令
		svgPathWsl := windowsPathToWslPath(svgPath)
		bitmapPathWsl := windowsPathToWslPath(bitmapPath)
		// 构造参数
		args := []string{"-f", "png", "-o", bitmapPathWsl, svgPathWsl}
		if bgColor != "" {
			args = append(args, "--background-color="+bgColor)
		}
		// 调用 wsl 运行 rsvg-convert
		cmd = exec.Command("wsl", append([]string{"rsvg-convert"}, args...)...)
	} else {
		// Linux/SCF 下直接用路径 (Linux 服务器需要安装 librsvg2-bin（Debian/Ubuntu）或 librsvg2-tools（CentOS/Fedora）)
		// 构造参数
		args := []string{"-f", "png", "-o", bitmapPath, svgPath, "-w", fmt.Sprint(targetSize), "-h", fmt.Sprint(targetSize)} // rsvg-convert 必须用 -f png
		if bgColor != "" {
			args = append(args, "--background-color="+bgColor) // 注意这里，把背景参数加到最后
		}
		zap.L().Debug("Running rsvg-convert", zap.Strings("args", args))
		cmd = exec.Command("/opt/bin/rsvg-convert", args...) // 提前把 rsvg-convert 工具放进 SCF 的层里了
	}

	output, err := cmd.CombinedOutput()

	if err != nil {
		zap.L().Error("cmd.Run() err:", zap.Error(err))
		return fmt.Errorf("convert failed: %v, output: %s", err, string(output))
	}
	// 第二步：如果 width/height 不等于 targetSize，需要进行等比缩放
	if width > 0 && height > 0 && (width != targetSize || height != targetSize) {
		// 打开生成的 PNG
		inFile, err := os.Open(bitmapPath)
		if err != nil {
			return err
		}
		img, err := png.Decode(inFile)
		inFile.Close()
		if err != nil {
			return err
		}

		// 按目标宽高缩放
		dst := image.NewRGBA(image.Rect(0, 0, width, height))
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), xdraw.Over, nil)
		// 写回文件
		outFile, err := os.Create(bitmapPath)
		if err != nil {
			return err
		}
		if err := png.Encode(outFile, dst); err != nil {
			outFile.Close()
			return err
		}
		outFile.Close()
	}

	// 第三步：根据 resourceType 决定是否需要二次转换
	// PNG 直接用 bitmapPath，不再 rename
	if strings.EqualFold(resourceType, "png") {
		return nil
	}
	// 其他格式：基于 PNG 再转
	return ConvertPngToOther(bitmapPath, bitmapPath, resourceType, bgColor)
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

// ConvertPngToOther 把 png 转换成 jpg、jpeg、webp格式的文件
func ConvertPngToOther(pngPath, outPath, resourceType, bgColor string) error {
	in, err := os.Open(pngPath)
	if err != nil {
		return err
	}
	defer in.Close()

	img, err := png.Decode(in)
	if err != nil {
		return err
	}

	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer out.Close()

	switch resourceType {
	case "jpg", "jpeg":
		// JPEG 不支持透明度：把 PNG 叠到统一底色上
		bg := ParseHexOrWhite(bgColor) // 默认白色
		rgba := ImageNewRGBAWithBG(img, bg)
		return jpeg.Encode(out, rgba, &jpeg.Options{Quality: 90})
	default:
		return fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}
