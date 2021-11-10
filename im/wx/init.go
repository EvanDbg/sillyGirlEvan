package wx

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/axgle/mahonia"
	"github.com/beego/beego/v2/adapter/httplib"
	"github.com/cdle/sillyGirl/core"
	"github.com/gin-gonic/gin"
)

var wx = core.NewBucket("wx")
var api_url = func() string {
	return wx.Get("api_url")
}
var robot_wxid = wx.Get("robot_wxid")

func sendTextMsg(pmsg *TextMsg) {
	pmsg.Msg = TrimHiddenCharacter(pmsg.Msg)
	if pmsg.Msg == "" {
		return
	}
	pmsg.Event = "SendTextMsg"
	pmsg.RobotWxid = robot_wxid
	req := httplib.Post(api_url())
	req.Header("Content-Type", "application/json")
	data, _ := json.Marshal(pmsg)
	enc := mahonia.NewEncoder("gbk")
	d := enc.ConvertString(string(data))
	d = regexp.MustCompile(`[\n\s]*\n[\s\n]*`).ReplaceAllString(d, "\n")
	req.Body(d)
	req.Response()
}

func sendMsg(pmsg *TextMsg) []byte {
	pmsg.Msg = "xxxx"
	pmsg.ToWxid = robot_wxid
	pmsg.RobotWxid = robot_wxid
	req := httplib.Post(api_url())
	req.Header("Content-Type", "application/json")
	data, _ := json.Marshal(pmsg)
	enc := mahonia.NewEncoder("gbk")
	d := enc.ConvertString(string(data))
	d = regexp.MustCompile(`[\n\s]*\n[\s\n]*`).ReplaceAllString(d, "\n")
	req.Body(d)
	rsp, _ := req.Response()
	x, _ := ioutil.ReadAll(rsp.Body)
	return x
}

func getNickname(nickname string, wxid string, grpWxid string) string {
	finalNickname := nickname
	hasGrpNick := false
	pmsg := TextMsg{
		Event:     "GetGroupMemberList",
		GroupWxid: grpWxid,
	}
	rs := sendMsg(&pmsg)
	grpMems := GroupMemberList{}
	json.Unmarshal(rs, &grpMems)
	mems := grpMems.Data
	for _, mem := range mems {
		if mem.Wxid == wxid {
			if (mem.Group_nickname != "") {
				finalNickname = mem.Group_nickname
				hasGrpNick = true
			}
			break
		}
	}
	if !hasGrpNick {
		pmsg = TextMsg{
			Event:     "GetFriendList",
		}
		rs = sendMsg(&pmsg)
		friends := FriendList{}
		json.Unmarshal(rs, &friends)
		frds := friends.Data
		for _, friend := range frds {
			if friend.Wxid == wxid {
				if (friend.Remark != "") {
					finalNickname = friend.Remark
				}
				break
			}
		}
	}
	return finalNickname
}

func TrimHiddenCharacter(originStr string) string {
	srcRunes := []rune(originStr)
	dstRunes := make([]rune, 0, len(srcRunes))
	for _, c := range srcRunes {
		if c >= 0 && c <= 31 && c != 10 {
			continue
		}
		if c == 127 {
			continue
		}
		dstRunes = append(dstRunes, c)
	}
	return string(dstRunes)
}

func sendOtherMsg(pmsg *OtherMsg) {
	if pmsg.Event == "" {
		pmsg.Event = "SendImageMsg"
	}
	pmsg.RobotWxid = robot_wxid
	req := httplib.Post(api_url())
	req.Header("Content-Type", "application/json")
	data, _ := json.Marshal(pmsg)
	req.Body(data)
	req.Response()
}

type TextMsg struct {
	Event      string `json:"event"`
	ToWxid     string `json:"to_wxid"`
	Msg        string `json:"msg"`
	RobotWxid  string `json:"robot_wxid"`
	GroupWxid  string `json:"group_wxid"`
	MemberWxid string `json:"member_wxid"`
}

type OtherMsg struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	Event      string `json:"event"`
	RobotWxid  string `json:"robot_wxid"`
	ToWxid     string `json:"to_wxid"`
	MemberWxid string `json:"member_wxid"`
	MemberName string `json:"member_name"`
	GroupWxid  string `json:"group_wxid"`
	Msg        Msg    `json:"msg"`
}

type PusherMsg struct {
	Subtitle    	string `json:"subtitle"`
	Title    			string `json:"title"`
	AppName    		string `json:"appName"`
	AppID    			string `json:"appID"`
	DeviceName    string `json:"deviceName"`
	Icon          string `json:"icon"`
	Message    		string `json:"message"`
}

type FriendList struct {
	Event         string   `json:"event"`
	Code          int      `json:"code"`
	Msg           string   `json:"msg"`
	Data          []Friend `json:"data"`
}

type Friend struct {
	Nickname    	string `json:"nickname"`
	Remark    		string `json:"remark"`
	WxNum    		  string `json:"wxNum"`
	Wxid    			string `json:"wxid"`
	Robot_wxid    string `json:"robot_wxid"`
}

type GroupMemberList struct {
	Event         string        `json:"event"`
	Code          int           `json:"code"`
	Msg           string        `json:"msg"`
	Data          []GroupMember `json:"data"`
}

type GroupMember struct {
	Count          int    `json:"count"`
	Group_nickname string `json:"group_nickname"`
	Nickname    	 string `json:"nickname"`
	WxNum    		   string `json:"wxNum"`
	Wxid    			 string `json:"wxid"`
	Identity       int    `json:"identity"`
}

type Msg struct {
	URL  string `json:"url"`
	Name string `json:"name"`
}

func init() {
	core.Pushs["wx"] = func(i interface{}, s string) {
		if robot_wxid != "" {
			pmsg := TextMsg{
				Msg:    s,
				ToWxid: fmt.Sprint(i),
			}
			sendTextMsg(&pmsg)
		}
	}
	core.GroupPushs["wx"] = func(i, j interface{}, s string) {
		to := fmt.Sprint(i) + "@chatroom"
		pmsg := TextMsg{
			ToWxid: to,
		}
		if j != nil && fmt.Sprint(j) != "" {
			pmsg.MemberWxid = fmt.Sprint(j)
		}
		for _, v := range regexp.MustCompile(`\[CQ:image,file=([^\[\]]+)\]`).FindAllStringSubmatch(s, -1) {
			s = strings.Replace(s, fmt.Sprintf(`[CQ:image,file=%s]`, v[1]), "", -1)
			data, err := os.ReadFile("data/images/" + v[1])
			if err == nil {
				add := regexp.MustCompile("(https.*)").FindString(string(data))
				if add != "" {
					pmsg := OtherMsg{
						ToWxid: to,
						Msg: Msg{
							URL:  relay(add),
							Name: name(add),
						},
					}
					defer sendOtherMsg(&pmsg)
				}
			}
		}
		s = regexp.MustCompile(`\[CQ:([^\[\]]+)\]`).ReplaceAllString(s, "")
		pmsg.Msg = s
		sendTextMsg(&pmsg)
	}
	core.Server.POST("/wx/receive", func(c *gin.Context) {
		data, _ := c.GetRawData()
		jms := JsonMsg{}
		json.Unmarshal(data, &jms)
		if jms.Event != "EventFriendMsg" && jms.Event != "EventGroupMsg" {
			c.JSON(200, map[string]string{"code": "1023"})
			return
		}
		// fmt.Println(jms.Type, "++++++++++++++++++++++------")
		if jms.Type != 1 && jms.Type != 3 { //|| jms.Type == 49
			// if jms.Type != 1 && jms.Type != 3 && jms.Type != 5 {
			return
		}
		if strings.Contains(fmt.Sprint(jms.Msg), `<type>57</type>`) {
			return
		}
		if jms.FinalFromWxid == jms.RobotWxid {
			c.JSON(200, map[string]string{"code": "1023"})
			return
		}
		listen := wx.Get("onGroups")
		if jms.Event == "EventGroupMsg" && listen != "" && !strings.Contains(listen, strings.Replace(fmt.Sprint(jms.FromWxid), "@chatroom", "", -1)) {
			c.JSON(200, map[string]string{"code": "1023"})
			return
		}
		if robot_wxid != jms.RobotWxid {
			robot_wxid = jms.RobotWxid
			wx.Set("robot_wxid", robot_wxid)
		}
		if wx.GetBool("keaimao_dynamic_ip", false) {
			ip, _ := c.RemoteIP()
			wx.Set("api_url", fmt.Sprintf("http://%s:%s", ip.String(), wx.Get("keaimao_port", "8080"))) //
		}
		//core.Senders <- &Sender{
		//	value: jms,
		//}

		msgBk := fmt.Sprintf("%s", jms.Msg)
		// msgBk := "[@at,nickname=EVAN,wxid=wxid_358hvbqajw2f12]  [@at,nickname=BBB,wxid=wxid_358hvbqajw2f13] [@at,nickname=CCC,wxid=wxid_358hvbqajw2f14]	 ceshiceshi"
		atOld := regexp.MustCompile(`\[@at,(.+?)\]`).FindAllStringSubmatch(msgBk, -1)
		atNew := regexp.MustCompile(`\[@at,nickname=(.+?),`).FindAllStringSubmatch(msgBk, -1)
		wxIds := regexp.MustCompile(`wxid=(.+?)\]`).FindAllStringSubmatch(msgBk, -1)
	
		pusherTitle := jms.FinalFromName
		if jms.FinalFromName != jms.FromName {
			pusherTitle = getNickname(pusherTitle, jms.FinalFromWxid, jms.FromWxid)
			pusherTitle = fmt.Sprintf("%s@%s", pusherTitle, jms.FromName)

			for i, v := range atOld {
				msgBk = strings.Replace(msgBk, v[0], fmt.Sprintf(`@%s `, getNickname(atNew[i][1], wxIds[i][1], jms.FromWxid)), -1)
			}
		}
		// fmt.Println(msgBk)
		// for i, v := range wxIds {
		// 	fmt.Println(i)
		// 	fmt.Println(v[1])
		// }	

		pusherMsg := PusherMsg{
			AppID: "com.tencent.xin",
			AppName: "",
			DeviceName: "",
			Title: pusherTitle,
			Subtitle: "",
			Icon: "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",
			Message: fmt.Sprintf("%s", msgBk),
		}
		apikey := wx.Get("apikey")
		dbCode := wx.Get("dbCode")
		if apikey != "" && dbCode != "" {
			req := httplib.Post(fmt.Sprintf("https://notifications-%s.restdb.io/rest/notifications", dbCode))
			req.Header("Content-Type", "application/json")
			req.Header("x-apikey", apikey)
			data, _ := json.Marshal(pusherMsg)
			req.Body(data)
			req.Response()
			c.JSON(200, map[string]string{"code": "999"})
			return
		}
		c.JSON(200, map[string]string{"code": "666"})
	})
	core.Server.GET("/relay", func(c *gin.Context) {
		url := c.Query("url")
		rsp, err := httplib.Get(url).Response()
		if err == nil {
			io.Copy(c.Writer, rsp.Body)
		}
	})
	core.Server.GET("/wximage", func(c *gin.Context) {
		c.Writer.Write([]byte{})
	})
}

var wxbase sync.Map

var myip = ""
var relaier = wx.Get("relaier")

func relay(url string) string {
	if wx.GetBool("relay_mode", false) == false {
		return url
	}
	if relaier != "" {
		return fmt.Sprintf(relaier, url)
	} else {
		if myip == "" || wx.GetBool("sillyGirl_dynamic_ip", false) == true {
			ip, _ := httplib.Get("https://imdraw.com/ip").String()
			if ip != "" {
				myip = ip
			}
		}
		return fmt.Sprintf("http://%s:%s/relay?url=%s", myip, wx.Get("relay_port", core.Bucket("sillyGirl").Get("port")), url) //"8002"
	}
}

type Sender struct {
	leixing int
	mtype   int
	deleted bool
	value   JsonMsg
	core.BaseSender
}

type JsonMsg struct {
	Event         string      `json:"event"`
	RobotWxid     string      `json:"robot_wxid"`
	RobotName     string      `json:"robot_name"`
	Type          int         `json:"type"`
	FromWxid      string      `json:"from_wxid"`
	FromName      string      `json:"from_name"`
	FinalFromWxid string      `json:"final_from_wxid"`
	FinalFromName string      `json:"final_from_name"`
	ToWxid        string      `json:"to_wxid"`
	Msg           interface{} `json:"msg"`
}

func (sender *Sender) GetContent() string {
	if sender.Content != "" {
		return sender.Content
	}
	switch sender.value.Msg.(type) {
	case int, int64, int32:
		return fmt.Sprintf("%d", sender.value.Msg)
	case float64:
		return fmt.Sprintf("%d", int(sender.value.Msg.(float64)))
	}
	return fmt.Sprint(sender.value.Msg)
}
func (sender *Sender) GetUserID() string {
	return sender.value.FinalFromWxid
}
func (sender *Sender) GetChatID() int {
	if strings.Contains(sender.value.FromWxid, "@chatroom") {
		return core.Int(strings.Replace(sender.value.FromWxid, "@chatroom", "", -1))
	} else {
		return 0
	}
}
func (sender *Sender) GetImType() string {
	return "wx"
}
func (sender *Sender) GetUsername() string {
	return sender.value.FinalFromName
}
func (sender *Sender) GetReplySenderUserID() int {
	if !sender.IsReply() {
		return 0
	}
	return 0
}
func (sender *Sender) IsAdmin() bool {
	return strings.Contains(wx.Get("masters"), fmt.Sprint(sender.GetUserID()))
}
func (sender *Sender) Reply(msgs ...interface{}) (int, error) {
	to := sender.value.FromWxid
	at := ""
	if to == "" {
		to = sender.value.FinalFromWxid
	} else {
		at = sender.value.FinalFromWxid
	}
	pmsg := TextMsg{
		ToWxid:     to,
		MemberWxid: at,
	}
	for _, item := range msgs {
		switch item.(type) {
		case string:
			pmsg.Msg = item.(string)
			images := []string{}
			for _, v := range regexp.MustCompile(`\[CQ:image,file=base64://([^\[\]]+)\]`).FindAllStringSubmatch(pmsg.Msg, -1) {
				images = append(images, v[1])
				pmsg.Msg = strings.Replace(pmsg.Msg, fmt.Sprintf(`[CQ:image,file=base64://%s]`, v[1]), "", -1)
			}
			// for _, image := range images {
			// 	wxbase
			// }
		case []byte:
			pmsg.Msg = string(item.([]byte))
		case core.ImageUrl:
			url := string(item.(core.ImageUrl))
			pmsg := OtherMsg{
				ToWxid:     to,
				MemberWxid: at,
				Msg: Msg{
					URL:  relay(url),
					Name: name(url),
				},
			}
			sendOtherMsg(&pmsg)
		}
	}
	if pmsg.Msg != "" {
		sendTextMsg(&pmsg)
	}
	return 0, nil
}

func name(str string) string {
	pr := "jpg"
	ss := regexp.MustCompile(`\.([A-Za-z0-9]+)$`).FindStringSubmatch(str)
	if len(ss) != 0 {
		pr = ss[1]
	}
	md5 := md5V(str)
	return md5 + "." + pr
}

func md5V(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

func (sender *Sender) Copy() core.Sender {
	new := reflect.Indirect(reflect.ValueOf(interface{}(sender))).Interface().(Sender)
	return &new
}
