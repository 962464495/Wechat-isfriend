package main

import (
	"fmt"
	"log"
	"os"
)

const (
	Debug    = false              //是否开启debug模式
	Duration = 2                  //请求腾讯接口的时间间隔
	Progress = 100                //进度条的长度
	DeviceId = "e356475304216396" //模拟设备ID 可以随意修改 但请保持长度不变
)

func main() {
	log.Println("[注意！！！]检测结果可能会引起不适。请您做好心理准备...")
	log.Println("[注意！！！]如出现检测的每位好友都将您删除的情况，请不要惊慌！")
	log.Println("[注意！！！]有可能是微信的问题哦！")
	log.Println("[注意！！！]一切以手机显示为准！！")
	log.Println("[注意！！！]为防止出现环境异常！之前没有登录过网页版微信的,请先前往https://wx.qq.com/登录一次")
	log.Println("[注意！！！]如果出现登录环境异常,请结束检测。您的账号被微信官方限制了！")
	log.Println("输入[y]继续进行检测,输入[n]结束检测")
	yes := ""
	fmt.Scanf("%s", &yes)
	switch yes {
	case "y":
		next()
		break
	case "n":
		log.Println("尊重您的选择！程序将在1s内退出！")
		os.Exit(3)
	default:
		log.Println("检测到其他输入！程序终止！")
	}
}
func next() {
	wx, err := NewWebwx()
	if err != nil {
		log.Printf("程序发生了致命性的错误: %s\n", err.Error())
		return
	}

	if err = wx.WaitForLogin(); err != nil {
		log.Println(err.Error())
		return
	}

	log.Println("登录中...")
	if err = wx.Login(); err != nil {
		log.Printf("登录失败: %s\n", err.Error())
		return
	}
	log.Println("登录成功")

	if err = wx.GetContact(); err != nil {
		log.Printf("获取联系人数据失败！: %s\n", err.Error())
		return
	}
	log.Printf("总共获取到[%d]联系人,其中普通好友[%d]人,预计总耗时[%d]s,开始查找\"好友\"\n", wx.Total, len(wx.MemberList), len(wx.MemberList)*Duration)

	wx.Getuesr()
	wx.Show()
	log.Println("感谢您的使用！ 按 Ctrl+C 退出程序")
	wx.WaitForExit()
}
