/*
# @Time : 2019-07-22 09:18
# @Author : smallForest
# @SoftWare : GoLand
*/
package main

import (
	"autoPay/application"
	"autoPay/conf"
	"encoding/json"
	"encoding/xml"
	"fmt"
	_ "github.com/astaxie/beego/httplib"
	"github.com/chanxuehong/wechat/mch/core"
	"github.com/chanxuehong/wechat/mch/pay"
	"github.com/gin-gonic/gin"
	"github.com/gogap/wechat/mp"
	"github.com/gogap/wechat/mp/jssdk"
	"github.com/gogap/wechat/mp/user/oauth2"
	_ "github.com/gogap/wechat/util"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
	_ "time"
)

type WXPayNotify struct {
	ReturnCode    string `xml:"return_code"`
	ReturnMsg     string `xml:"return_msg"`
	Appid         string `xml:"appid"`
	MchID         string `xml:"mch_id"`
	DeviceInfo    string `xml:"device_info"`
	NonceStr      string `xml:"nonce_str"`
	Sign          string `xml:"sign"`
	ResultCode    string `xml:"result_code"`
	ErrCode       string `xml:"err_code"`
	ErrCodeDes    string `xml:"err_code_des"`
	Openid        string `xml:"openid"`
	IsSubscribe   string `xml:"is_subscribe"`
	TradeType     string `xml:"trade_type"`
	BankType      string `xml:"bank_type"`
	TotalFee      int64  `xml:"total_fee"`
	FeeType       string `xml:"fee_type"`
	CashFee       int64  `xml:"cash_fee"`
	CashFeeType   string `xml:"cash_fee_type"`
	CouponFee     int64  `xml:"coupon_fee"`
	CouponCount   int64  `xml:"coupon_count"`
	CouponID0     string `xml:"coupon_id_0"`
	CouponFee0    int64  `xml:"coupon_fee_0"`
	TransactionID string `xml:"transaction_id"`
	OutTradeNo    string `xml:"out_trade_no"`
	Attach        string `xml:"attach"`
	TimeEnd       string `xml:"time_end"`
}
type AutoOrder struct {
	ID         int
	Openid     string
	Fee        int
	TransId    string
	Status     int
	CreateTime int
	PayTime    int
	Fees       float32
	PayTimes   string
}

func (AutoOrder) TableName() string {
	return "auto_order"
}

// to grant authorization
type Ga struct {
	gorm.Model
	Openid   string `gorm:"unique"`//唯一
	Userinfo string `gorm:"size:2000"`
}

var oauth2Config = oauth2.NewOAuth2Config(
	application.Appid,
	application.Appsecret,
	application.Domain+"/letusgo/chongdingxiang",
	"snsapi_userinfo")

func main() {

	// 声明数据库连接
	db := application.Mysql()
	defer db.Close()
	// 开启log
	db.LogMode(true)
	// 连接池数量
	db.DB().SetMaxIdleConns(10)

	router := gin.Default()

	router.GET("/letusgo/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	// 以上都是测试

	router.LoadHTMLGlob("templates/*")
	// router.LoadHTMLFiles("templates/template1.html", "templates/template2.html")
	handlers := func(c *gin.Context) {
		openid := c.Query("id")
		fmt.Println(openid)
		if openid == "" {
			// 将路由设置进state 好跳转回来
			AuthCodeURL := oauth2Config.AuthCodeURL("/letusgo/index")
			c.Redirect(302, AuthCodeURL)
		}
		// 去数据库查询授权记录
		var ga Ga
		db.AutoMigrate(&ga)
		db.Where("openid=?", openid).First(&ga)
		fmt.Println(ga)
		// 没有查询到授权记录 去授权
		if ga.ID == 0 {
			// 将路由设置进state 好跳转回来
			AuthCodeURL := oauth2Config.AuthCodeURL("/letusgo/index")
			c.Redirect(302, AuthCodeURL)
		}

		// 获取jssdk参数信息
		var AccessTokenServer = mp.NewDefaultAccessTokenServer(application.Appid, application.Appsecret, nil)
		var TicketServer = jssdk.NewDefaultTicketServer(AccessTokenServer, nil)
		if ticket, err := TicketServer.Ticket(); err != nil {
			fmt.Println(err)
			return
		} else {
			nonceStr := application.GetRandomString(16)
			time1 := application.CurrentTimestamp()
			timestamp := strconv.Itoa(time1)
			url := application.Domain + "/letusgo/index"
			signature := jssdk.WXConfigSign(ticket, nonceStr, timestamp, url)

			// 获取已经支付的列表
			db.AutoMigrate(&AutoOrder{})
			autoOrders := make([]AutoOrder, 0)
			db.Where("status=?", 1).Where("openid=?", openid).Order("id desc").Find(&autoOrders)
			for k, v := range autoOrders {
				autoOrders[k].Fees = application.Chu(v.Fee)
				autoOrders[k].PayTimes = time.Unix(int64(v.PayTime), 0).Format("2006/01/02 15:04:05")
			}

			c.HTML(http.StatusOK, "autoPay.tmpl", gin.H{
				"title":     "pay everyday",
				"openId":    openid,
				"appId":     application.Appid,
				"timestamp": time1,
				"nonceStr":  nonceStr,
				"signature": signature,
				"list":      autoOrders,
			})
		}

	}
	router.GET("/letusgo/index", handlers)
	router.GET("/letusgo/chongdingxiang", func(c *gin.Context) {
		fmt.Println(c.Request.URL)
		code := c.Query("code")
		state := c.Query("state")
		if code == "" {
			fmt.Println("客户禁止授权")
			return
		}
		fmt.Println(state)
		var oauth2Token oauth2.OAuth2Token
		var oauth2Client = oauth2.Client{
			oauth2Config,
			&oauth2Token,
			nil,
		}
		_, err := oauth2Client.Exchange(code)
		if err != nil {
			fmt.Println(err)
			return
		}

		userinfo, err := oauth2Client.UserInfo(oauth2.Language_zh_CN)
		if err != nil {
			fmt.Println(err)
			return
		}
		// 将userinfo存储数据库
		b, err := json.Marshal(userinfo)
		if err != nil {
			fmt.Println(err)
			return
		}
		ga := Ga{Openid: userinfo.OpenId, Userinfo: string(b)}
		db.AutoMigrate(&ga)
		db.Create(&ga)

		fmt.Println(userinfo)
		// 跳转到index页面
		c.Redirect(302, state+"?id="+userinfo.OpenId)

	})
	// 创建待支付订单
	router.POST("/letusgo/pay", func(c *gin.Context) {
		// 获取用的openID
		openid := c.PostForm("id")
		fmt.Println("openid", openid)
		trans_id := "AT" + application.GetRandomString(17)
		fee := int64(application.GenerateRangeNum(1, 100))
		uni := pay.UnifiedOrderRequest{
			Body:           "知识付费",
			OutTradeNo:     trans_id,
			TotalFee:       fee,
			SpbillCreateIP: c.ClientIP(),
			NotifyURL:      application.Domain + "/letusgo/NotifyURL",
			TradeType:      "JSAPI",
			OpenId:         openid,
		}
		var client = core.NewClient(application.Appid, application.Mchid, application.Key, nil)
		if response, err := pay.UnifiedOrder2(client, &uni); err != nil {
			fmt.Println(err)
			return
		} else {
			timestamp := strconv.Itoa(application.CurrentTimestamp())
			nonceStr := application.GetRandomString(16)
			package_str := "prepay_id=" + response.PrepayId
			signType := "MD5"
			paySign := core.JsapiSign(application.Appid, timestamp, nonceStr, package_str, signType, application.Key)

			// 插入订单
			order := AutoOrder{
				Openid:     openid,
				Fee:        int(fee),
				TransId:    trans_id,
				Status:     0,
				CreateTime: int(time.Now().Unix()),
				PayTime:    0,
			}
			db.Create(&order)
			c.JSON(http.StatusOK, gin.H{
				"timestamp":   timestamp,
				"nonceStr":    nonceStr,
				"package_str": package_str,
				"signType":    signType,
				"paySign":     paySign,
			})
		}
	})
	router.POST("/letusgo/NotifyURL", func(c *gin.Context) {
		res := c.Request.Body
		fmt.Println("回调结果", res)
		bodydata, err := ioutil.ReadAll(res)
		if err != nil {
			fmt.Println(err)
		}
		var wxn WXPayNotify
		err = xml.Unmarshal(bodydata, &wxn)
		if err != nil {
			fmt.Println(err)
		}

		// 按照查询
		fmt.Println(wxn)
		// 通过订单号获取信息
		var autoOrder AutoOrder
		db.Where("trans_id=?", wxn.OutTradeNo).First(&autoOrder)
		// 未支付修改成支付
		if autoOrder.Status == 0 {
			autoOrder.Status = 1
			autoOrder.PayTime = int(time.Now().Unix())
			db.Save(&autoOrder)
		}
		c.XML(200, gin.H{"return_code": "SUCCESS", "return_msg": "OK"})
	})
	// 静态文件服务 微信域名认证
	router.StaticFile("/MP_verify_6mlClfsaZU0IuAW9.txt", "./MP_verify_6mlClfsaZU0IuAW9.txt")

	_ = router.Run(conf.Run().Section("app").Key("start_listen_port").String())
}
