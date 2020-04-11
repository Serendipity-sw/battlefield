package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/go-vgo/robotgo"
	"github.com/kbinani/screenshot"
	"github.com/smtc/glog"
	"github.com/swgloomy/gutil"
	"image/png"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	screenWidth  int
	screenHeight int
	debugFlag    = flag.Bool("d", false, "debug mode") //是否为调试模式
	threadLock   sync.Mutex
)

func main() {
	flag.Parse()

	gutil.LogInit(*debugFlag, "./logs")

	//获取当前显示器大小
	screenWidth, screenHeight = robotgo.GetScaleSize()

	go script()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGTERM)
	<-c

	serviceExit()
}

func serviceExit() {
	glog.Close()
	os.Exit(0)
}

func script() {
time.Sleep(3 * time.Second)
	go temporarilyLeave()
	go queue()
}

//暂离功能
func temporarilyLeave() {
	for true {
		threadLock.Lock()
		//线程等待10秒 使窗口置顶操作正常进行
		time.Sleep(10 * time.Second)

		//将鼠标移动到屏幕中央
		robotgo.MoveMouse(screenWidth/2, screenHeight/2)
		//鼠标左键按住操作
		robotgo.MouseToggle("down")
		//鼠标左键按住向右拖动半个屏幕
		robotgo.DragMouse(screenWidth, screenHeight/2)
		//鼠标左键松开操作 结束视角转向
		robotgo.MouseToggle("up")

		time.Sleep(2 * time.Second)

		//人物动作事件开始
		//操作人物往前移动
		robotgo.KeyToggle("w", "down")
		//移动三秒钟
		time.Sleep(3 * time.Second)
		//结束人物移动
		robotgo.KeyToggle("w", "up")

		time.Sleep(3 * time.Second)
		//操作人物往右移动
		robotgo.KeyToggle("d", "down")
		//移动三秒钟
		time.Sleep(3 * time.Second)
		//结束人物移动
		robotgo.KeyToggle("d", "up")

		time.Sleep(2 * time.Second)
		//控制人物跳动一次
		robotgo.KeyTap("space")
		//等待人物落地.为了防止人物卡顿,等待3秒
		time.Sleep(3 * time.Second)

		//施放一次快捷键技能 对自己  ctrl + s  最好为骑马快捷键或群疗快捷键 或者 3分钟内的大招 凡是一切不需要选中目标施放的技能都可以
		robotgo.KeyTap("s", "lctrl")

		threadLock.Unlock()

		time.Sleep(2 * time.Minute)
	}
}

//排队
func queue() {
	var errMessage string

	type resultStruct struct {
		Words string `json:"words"`
	}

	for true {
		var (
			wordArray []resultStruct
			bo        = false
		)

		//百度token 获取
		baiduToken := baiduAccessToken()

		//屏幕截图
		fileName := captureRect()

		//百度图片文字识别
		words_result := characterRecognition(baiduToken, fileName)

		err := json.Unmarshal([]byte(words_result), &wordArray)
		if err != nil {
			errMessage = fmt.Sprintf("图片文字解析反序列化失败! result: %s err: %s \n", words_result, err.Error())
			fmt.Print(errMessage)
			glog.Error(errMessage)
		} else {
			for _, item := range wordArray {
				if strings.Index(item.Words, "进入战场") > -1 {
					bo = true
				}
			}
			if bo {
				break
			}
		}

		err = os.Remove(fileName)
		if err != nil {
			errMessage = fmt.Sprintf("图片删除失败! fileName: %s err: %s \n", fileName, err.Error())
			fmt.Print(errMessage)
			glog.Error(errMessage)
		}

		time.Sleep(1 * time.Minute)
	}

	threadLock.Lock()
	robotgo.KeyTap("enter")
	time.Sleep(20 * time.Second)
	threadLock.Unlock()
}

func isLoginInterface() bool {
	var errMessage string

	//百度token 获取
	baiduToken := baiduAccessToken()

	//屏幕截图
	fileName := captureRect()

	//百度图片文字识别
	words_result := characterRecognition(baiduToken, fileName)

	type resultStruct struct {
		Words string `json:"words"`
	}

	var (
		wordArray []resultStruct
		bo        = false
	)

	err := json.Unmarshal([]byte(words_result), &wordArray)
	if err != nil {
		errMessage = fmt.Sprintf("图片文字解析反序列化失败! result: %s err: %s \n", words_result, err.Error())
		fmt.Print(errMessage)
		glog.Error(errMessage)
		return false
	}

	for _, item := range wordArray {
		if strings.Index(item.Words, "进入魔兽世界") > -1 {
			bo = true
		}
	}
	return bo
}

func captureRect() string {
	var errMessage string
	//获取屏幕 进行屏幕截图
	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		errMessage = fmt.Sprintf("屏幕截图失败! err: %s \n", err.Error())
		fmt.Print(errMessage)
		glog.Error(errMessage)
		return ""
	}
	fileName := fmt.Sprintf("%dx%d-%d.png", bounds.Dx(), bounds.Dy(), time.Now().Unix())
	file, err := os.Create(fileName)
	if err != nil {
		errMessage = fmt.Sprintf("创建图片文件失败! fileName: %s err: %s \n", fileName, err.Error())
		fmt.Print(errMessage)
		glog.Error(errMessage)
		return ""
	}
	defer func() {
		err = file.Close()
		if err != nil {
			errMessage = fmt.Sprintf("图片文件关闭失败! fileName: %s err: %s \n", fileName, err.Error())
			fmt.Print(errMessage)
			glog.Error(errMessage)
			return
		}
	}()
	err = png.Encode(file, img)
	if err != nil {
		errMessage = fmt.Sprintf("图片文件写入失败! fileName: %s err: %s \n", fileName, err.Error())
		fmt.Print(errMessage)
		glog.Error(errMessage)
		return ""
	}
	return fileName
}

func baiduAccessToken() string {
	var errMessage string
	httpUrl := fmt.Sprintf("https://aip.baidubce.com/oauth/2.0/token?grant_type=client_credentials&client_id=%s&client_secret=%s", "CZRPiRucoN6ViwcGuH30N00H", "7H4jqYzcfAXorFuDlxGNdoIx3ixsoNG3")
	result, err := http.PostForm(httpUrl, nil)
	if err != nil {
		errMessage = fmt.Sprintf("百度授权token获取失败! http: %s err: %s \n", httpUrl, err.Error())
		fmt.Print(errMessage)
		glog.Error(errMessage)
		return ""
	}
	var resultMap = make(map[string]interface{})
	defer func() {
		err = result.Body.Close()
		if err != nil {
			errMessage = fmt.Sprintf("百度授权token请求返回体关闭失败! http: %s err: %s \n", httpUrl, err.Error())
			fmt.Print(errMessage)
			glog.Error(errMessage)
		}
	}()
	resultByte, err := ioutil.ReadAll(result.Body)
	if err != nil {
		errMessage = fmt.Sprintf("百度授权token返回体读取失败! http: %s err: %s \n", httpUrl, err.Error())
		fmt.Print(errMessage)
		glog.Error(errMessage)
		return ""
	}
	resultString := string(resultByte)
	lastIndex := strings.LastIndex(resultString, "}")
	resultString = resultString[0 : lastIndex+1]
	err = json.Unmarshal([]byte(resultString), &resultMap)
	if err != nil {
		errMessage = fmt.Sprintf("百度授权token返回结构体json序列化失败! resultByte: %s httpUrl: %s err: %s \n", string(resultByte), httpUrl, err.Error())
		fmt.Print(errMessage)
		glog.Error(errMessage)
		return ""
	}

	access_token, ok := resultMap["access_token"]
	if ok {
		return access_token.(string)
	} else {
		return ""
	}
}

func characterRecognition(token string, fileName string) string {
	var errMessage string
	httpUrl := fmt.Sprintf("https://aip.baidubce.com/rest/2.0/ocr/v1/general_basic?access_token=%s", token)
	urlValues := url.Values{}

	f, err := os.Open(fileName)
	if err != nil {
		errMessage = fmt.Sprintf("图片文件打开失败! fileName: %s err: %s \n", fileName, err.Error())
		fmt.Print(errMessage)
		glog.Error(errMessage)
		return ""
	}
	reader := bufio.NewReader(f)
	content, _ := ioutil.ReadAll(reader)
	encoded := base64.StdEncoding.EncodeToString(content)

	urlValues.Add("image", encoded)

	result, err := http.PostForm(httpUrl, urlValues)
	if err != nil {
		errMessage = fmt.Sprintf("百度图片识别请求失败! httpUrl: %s urlValues: %v err: %s \n", httpUrl, urlValues, err.Error())
		fmt.Print(errMessage)
		glog.Error(errMessage)
		return ""
	}
	defer func() {
		err = result.Body.Close()
		if err != nil {
			errMessage = fmt.Sprintf("百度图片识别返回结构体关闭失败! httpUrl: %s urlValues: %v err: %s \n", httpUrl, urlValues, err.Error())
			fmt.Print(errMessage)
			glog.Error(errMessage)
			return
		}
	}()
	resultByte, err := ioutil.ReadAll(result.Body)
	if err != nil {
		errMessage = fmt.Sprintf("百度图片识别返回结构体读取失败! httpUrl: %s urlValues: %v err: %s \n", httpUrl, urlValues, err.Error())
		fmt.Print(errMessage)
		glog.Error(errMessage)
		return ""
	}
	var resultMap = make(map[string]interface{})
	resultString := string(resultByte)
	lastIndex := strings.LastIndex(resultString, "}")
	resultString = resultString[0 : lastIndex+1]
	err = json.Unmarshal([]byte(resultString), &resultMap)
	if err != nil {
		errMessage = fmt.Sprintf("百度图片识别返回结构体序列化失败! resultByte: %s httpUrl: %s urlValues: %v err: %s \n", string(resultByte), httpUrl, urlValues, err.Error())
		fmt.Print(errMessage)
		glog.Error(errMessage)
		return ""
	}
	words_result, ok := resultMap["words_result"]
	if ok {
		jsonByte, _ := json.Marshal(words_result)
		return string(jsonByte)
	} else {
		return ""
	}
}
