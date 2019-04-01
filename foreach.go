package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego/config"
	"log"
	"time"
)

var (
	DelCount = 0
)
//获取所有的联系人
func (this *Webwx) GetContact() (err error) {
	name, resp := "webwxgetcontact", new(MemberResp)
	apiUri := fmt.Sprintf("%s/%s?pass_ticket=%s&skey=%s&r=%s", this.BaseUri, name, this.Request.PassTicket, this.Request.Skey, time.Now().Unix())
	if err = this.send(apiUri, name, nil, resp); err != nil {
		return
	}

	this.MemberList, this.Total = make([]*Member, 0, resp.MemberCount/5*2), resp.MemberCount
	for i := 0; i < this.Total; i++ {
		if resp.MemberList[i].IsNormal() {
			this.MemberList = append(this.MemberList, resp.MemberList[i])
		}
	}

	return
}

//发送消息

func (this *Webwx) SendMessage(user *Member, my string) {
	time := config.ToString(time.Now().Unix())
	message := Messages{
		1,
		Message,
		my,
		user.UserName,
		time,
		time,
		"",
	}

	send := SendMessge{
		this.Request,
		message,
		0,
	}
	data, err := json.Marshal(send)
	if err != nil {
		return
	}
	name, resp := "webwxsendmsg", new(MessageResponse)
	apiUri := fmt.Sprintf("%s/%s?pass_ticket=%s", this.BaseUri, name, this.Request.PassTicket)
	if err = this.send(apiUri, name, bytes.NewReader(data), resp); err != nil {
		log.Printf("检测用户[%s]时出错,错误信息:%s", user.NickName, err)
		return
	}
	this.RevokeMessage(resp.MsgID, Myself, resp.LocalID, user)

}

//撤回消息
func (this *Webwx) RevokeMessage(msg_id string, user_id string, local_msg_id string, user *Member) {
	testerevoke := RevokeMessge{
		this.Request,
		msg_id,
		user_id,
		local_msg_id,
	}
	data, err := json.Marshal(testerevoke)
	if err != nil {
		return
	}
	name, resp := "webwxrevokemsg", new(RevokeMessgeResponse)
	apiUri := fmt.Sprintf("%s/%s", this.BaseUri, name)
	if err = this.send(apiUri, name, bytes.NewReader(data), resp); err != nil {

	}
	if resp.BaseResponse.Ret == -1 {
		if user.RemarkName != "" {
			log.Printf("用户名为:%s,备注名为:%s,的用户已经将您删除好友", user.NickName, user.RemarkName)
			DelCount++
		} else {
			log.Printf("用户名为:%s的用户已经将您删除好友", user.NickName)
			DelCount++
		}
	} else {
		if user.RemarkName != "" {
			log.Printf("用户名为:%s,备注名为:%s的好友,校验完毕！", user.NickName, user.RemarkName)
		} else {
			log.Printf("用户名为:%s的好友,校验完毕！", user.NickName)
		}
	}

}

//获取所有的还有列表 抛出公众号以及其他没用的
func (this *Webwx) Getuesr() {
	total := len(this.MemberList)
	if total == 0 {
		return
	}
	names, users, namesMap := make([]string, 0), make([]User, 0), make(map[string]*Member)
	for i, member := range this.MemberList {
		if len(this.ChatRoomName) == 0 {
			users = append(users, User{
				UserName: member.UserName,
			})
		}
		names, namesMap[member.UserName] = append(names, member.UserName), member
		log.Printf("正在校验%s", member.NickName)
		this.SendMessage(member, Myself)
		if i <= total {
			log.Printf("为了防止腾讯检测频繁，程序等待 %ds 后将继续检测，请耐心等待...\n", Duration)
			time.Sleep(time.Duration(Duration) * time.Second)
		}
		this.progress(i+1, total, DelCount)
	}
	this.progress(total, total, DelCount)
	return

}
