package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	//成功的标识
	Success = 0
	//Message = "test message"
	//利用微信漏洞批量发送的消息
	Message = "జ్ఞా"
)

var (
	//用来记录自己的ID
	Myself string
	//一些腾讯自己的好友标识  用来筛选好友
	SpecialUsers = []string{
		"newsapp", "fmessage", "filehelper", "weibo", "qqmail",
		"tmessage", "qmessage", "qqsync", "floatbottle", "lbsapp",
		"shakeapp", "medianote", "qqfriend", "readerapp", "blogapp",
		"facebookapp", "masssendapp", "meishiapp", "feedsapp", "voip",
		"blogappweixin", "weixin", "brandsessionholder", "weixinreminder", "wxid_novlwrv3lqwv11",
		"gh_22b87fa7cb3c", "officialaccounts", "notification_messages", "wxitil", "userexperience_alarm",
	}
)

// 请求的构造体
type Request struct {
	BaseRequest *BaseRequest

	MemberCount int    `json:",omitempty"`
	MemberList  []User `json:",omitempty"`
	Topic       string `json:",omitempty"`

	ChatRoomName  string `json:",omitempty"`
	DelMemberList string `json:",omitempty"`
	AddMemberList string `json:",omitempty"`
}

type BaseRequest struct {
	XMLName xml.Name `xml:"error",json:"-"`

	Ret        int    `xml:"ret",json:"-"`
	Message    string `xml:"message",json:"-"`
	Skey       string `xml:"skey"`
	Wxsid      string `xml:"wxsid",json:"Sid"`
	Wxuin      int    `xml:"wxuin",json:"Uin"`
	PassTicket string `xml:"pass_ticket",json:"-"`

	DeviceID string `xml:"-"`
}

type Caller interface {
	IsSuccess() bool
	Error() error
}

type Response struct {
	BaseResponse *BaseResponse
}

//发送消息的构造体
type SendMessge struct {
	BaseRequest *BaseRequest
	Msg         Messages
	Scene       int
}

//消息的构造体
type Messages struct {
	Type         int
	Content      string
	FromUserName string
	ToUserName   string
	LocalID      string
	ClientMsgId  string
	MediaId      string
}

//发送消息的返回值的构造体
type MessageResponse struct {
	Response
	MsgID   string
	LocalID string
}

//撤回消息的构造体
type RevokeMessge struct {
	BaseRequest *BaseRequest
	SvrMsgId    string
	ToUserName  string
	ClientMsgId string
}

//用来接收撤回消息的返回值的构造体
type RevokeMessgeResponse struct {
	Response
	Introduction string
	SysWording   string
}

//判断是否成功
func (this *Response) IsSuccess() bool {
	return this.BaseResponse.Ret == Success
}

func (this *Response) Error() error {
	return fmt.Errorf("message:[%s]", this.BaseResponse.ErrMsg)
}

type BaseResponse struct {
	Ret    int
	ErrMsg string
}

type InitResp struct {
	Response
	User User
}

type User struct {
	UserName string
}

type MemberResp struct {
	Response
	MemberCount  int
	ChatRoomName string
	MemberList   []*Member
}

type Member struct {
	UserName     string
	NickName     string
	RemarkName   string
	VerifyFlag   int
	MemberStatus int
}

func (this *Member) IsOnceFriend() bool {
	return this.MemberStatus == 4
}

//判断 好友类型
func (this *Member) IsNormal() bool {
	return this.VerifyFlag&8 == 0 && // 公众号/服务号
		!strings.HasPrefix(this.UserName, "@@") && // 群聊
		this.UserName != Myself && // 自己
		!this.IsSpecail() // 特殊账号
}

func (this *Member) IsSpecail() bool {
	for i, count := 0, len(SpecialUsers); i < count; i++ {
		if this.UserName == SpecialUsers[i] {
			return true
		}
	}
	return false
}

type Webwx struct {
	Client  *http.Client
	Request *BaseRequest

	CurrentDir  string
	QRImagePath string

	RedirectUri  string
	BaseUri      string
	ChatRoomName string // 用于查找好友的群号

	Total      int       // 好友总数
	MemberList []*Member // 普通好友

	OnceFriends []string
}

//用来接收返回值的二维码并创建文件
func NewWebwx() (wx *Webwx, err error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return
	}

	transport := *(http.DefaultTransport.(*http.Transport))
	transport.ResponseHeaderTimeout = 1 * time.Minute
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	wx = &Webwx{
		Client: &http.Client{
			Transport: &transport,
			Jar:       jar,
			Timeout:   1 * time.Minute,
		},
		Request: new(BaseRequest),

		CurrentDir:  currentDir,
		QRImagePath: filepath.Join(currentDir, "qrcode.jpg"),
	}
	return
}

func newClient() (client *http.Client) {
	transport := *(http.DefaultTransport.(*http.Transport))
	transport.ResponseHeaderTimeout = 1 * time.Minute
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Panicln(err.Error())
	}

	client = &http.Client{
		Transport: &transport,
		Jar:       jar,
		Timeout:   1 * time.Minute,
	}
	return
}

func createFile(name string, data []byte, isAppend bool) (err error) {
	oflag := os.O_CREATE | os.O_WRONLY
	if isAppend {
		oflag |= os.O_APPEND
	} else {
		oflag |= os.O_TRUNC
	}

	file, err := os.OpenFile(name, oflag, 0600)
	if err != nil {
		return
	}
	defer file.Close()

	_, err = file.Write(data)
	return
}

func findData(data, prefix, suffix string) (result string, err error) {
	index := strings.Index(data, prefix)
	if index == -1 {
		err = fmt.Errorf("本程序已无法处理接口返回的新格式的数据:[%s]", data)
		return
	}
	index += len(prefix)

	end := strings.Index(data[index:], suffix)
	if end == -1 {
		err = fmt.Errorf("本程序已无法处理接口返回的新格式的数据:[%s]", data)
		return
	}

	result = data[index : index+end]
	return
}

//网络请求的方法
func (this *Webwx) send(apiUri, name string, body io.Reader, call Caller) (err error) {
	method := "GET"
	if body != nil {
		method = "POST"
	}
	req, err := http.NewRequest(method, apiUri, body)
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	resp, err := this.Client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	reader := resp.Body.(io.Reader)
	if Debug {
		var data []byte
		data, err = ioutil.ReadAll(reader)
		if err != nil {
			return
		}

		if err = createFile(filepath.Join(this.CurrentDir, name+".json"), data, strings.HasSuffix(name, "member")); err != nil {
			return
		}
		reader = bytes.NewReader(data)
	}

	if err = json.NewDecoder(reader).Decode(call); err != nil {
		return
	}

	if !call.IsSuccess() {
		return call.Error()
	}
	return
}

func (this *Webwx) search(members []*Member, namesMap map[string]*Member) {
	for _, member := range members {
		if member.IsOnceFriend() {
			m, ok := namesMap[member.UserName]
			if !ok {
				m = member
			}
			this.OnceFriends = append(this.OnceFriends, fmt.Sprintf("昵称:[%s], 备注:[%s]", m.NickName, m.RemarkName))
		}
	}
}

func (this *Webwx) progress(current, total int, delcount int) {
	done := current * Progress / total
	log.Printf("已完成[%d]位好友的检测,共有[%d]位好友已将您删除\n", current, delcount)
	log.Println("[" + strings.Repeat("#", done) + strings.Repeat("-", Progress-done) + "]")
}

func (this *Webwx) Show() {
	if DelCount == 0 {
		log.Println("恭喜您，没有一位好友将您删除")
	} else {
		log.Printf("共有%d位用户将您删除好友", DelCount)
	}
	fmt.Println("---------------------------------------------")
	fmt.Println("-------这里的检测结果有误差！以手机显示为主-------")
	fmt.Println("---如不需要你再次检测,请在手机端点击退出网页登录---")
	fmt.Println("---------------------------------------------")
	return
}

func (this *Webwx) WaitForExit() os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGTERM)
	return <-c
}
