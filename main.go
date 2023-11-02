package main

import (
	"errors"
	"fmt"
	"github.com/davidbyttow/govips/v2/vips"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	client = &http.Client{}
)

func LogTimeCost(name string, timestamp time.Time) {
	fmt.Println(name, ": Process Cost", time.Since(timestamp))
}

func main() {
	vips.Startup(nil)
	defer vips.Shutdown()
	// start a gin server and handle /_next/image path
	r := gin.Default()
	r.GET("/_next/image", handleImage)
	r.Run()
	gin.New()
}

func handleImage(c *gin.Context) {
	defer LogTimeCost("ALL", time.Now())
	url := c.Query("url")
	url = strings.Replace(url, "https://cdn.crushon.ai", "http://cdn.crushon.ai.s3-ap-southeast-1.amazonaws.com", 1)
	transformOption := transformOption(c.Query("w"), c.Query("h"), c.Query("q"))
	fmt.Println("url:", url, "option:", transformOption)
	image, err := fetchImage(url, map[string]string{
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/97.0.4692.71 Safari/537.36", //c.GetHeader("User-Agent"),
		"Accept":          "*/*",
		"Accept-Encoding": "gzip, deflate, br",
		"Referer":         c.GetHeader("https://crushon.ai"),
	})
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	result, err := processImage(image, transformOption)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.Data(http.StatusOK, "image/webp", result)
}

type TransformOption struct {
	width   int
	height  *int
	quality *int
}

func (t *TransformOption) Width() int {
	return t.width
}

func (t *TransformOption) Height() *int {
	return t.height
}

func (t *TransformOption) Quality() int {
	if t.quality == nil {
		return 75
	}
	return *t.quality
}

func processImage(buffer []byte, option TransformOption) (result []byte, err error) {
	defer LogTimeCost("processImage", time.Now())
	inputImage, err := vips.NewImageFromBuffer(buffer)
	if err != nil {
		fmt.Println("Error creating image from reader:", err)
		return
	}
	defer inputImage.Close()

	_ = inputImage.AutoRotate()
	// resize with aspect ratio
	if scale := float64(option.width) / float64(inputImage.Width()); scale < 1 {
		fmt.Println("Resize image with scale:", scale, inputImage.Width())
		_ = inputImage.Resize(scale, vips.KernelAuto)
	}
	// convert
	ep := vips.NewWebpExportParams()
	ep.Quality = option.Quality()

	imageBytes, _, err := inputImage.ExportWebp(ep)
	if err != nil {
		fmt.Println("Error exporting image to webp:", err)
		return
	}
	return imageBytes, nil
}

func transformOption(width string, height string, quality string) (option TransformOption) {
	option = TransformOption{}
	if width != "" {
		option.width, _ = strconv.Atoi(width)
	}
	if height != "" {
		if value, err := strconv.Atoi(height); err == nil {
			option.height = &value
		}
	}
	if quality != "" {
		if value, err := strconv.Atoi(quality); err == nil {
			option.quality = &value
		}
	}
	return
}

func fetchImage(url string, header map[string]string) (buffer []byte, err error) {
	defer LogTimeCost("fetchImage", time.Now())
	// 发起HTTP GET请求获取图像数据
	// Create an HTTP GET request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error creating the request:", err)
		return
	}
	// Set custom headers
	for key, value := range header {
		req.Header.Set(key, value)
	}
	response, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending the request:", err)
		return
	}
	defer response.Body.Close()
	// 检查HTTP响应状态码
	if response.StatusCode != http.StatusOK {
		fmt.Println("HTTP response error:", response.Status, response)
		return nil, errors.New(response.Status)
	}

	buffer, err = io.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}
	return
}
