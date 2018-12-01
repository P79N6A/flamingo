package main

import (
	"code.bean.com/flamingo/config"
	"code.bean.com/flamingo/handler"
	"code.bean.com/flamingo/model"
	"code.byted.org/gin/ginex"
	"code.byted.org/gopkg/logs"
	"github.com/gin-gonic/gin"
)

var (
	e *ginex.Engine
)

func main() {
	router := gin.Default()
	config.Init("./conf/flamingo.json")
	model.Init()
	logs.Info("init model finished")
	handler.Init()
	router.Static("/templates/css", "templates/css")
	router.Static("/templates/js", "templates/js")
	router.Static("/templates/icons", "templates/icons")
	router.Static("/templates/images", "templates/images")
	router.LoadHTMLGlob("templates/*.html")
	handler.RegisterHandler(router)

	// i := 0
	// // router.LoadHTMLFiles("templates/bangdan.html", "templates/huanjing.html", "templates/waimai.html", "templates/jquery-3.2.1.min.js")
	// router.GET("/wx/wx_conn", func(c *gin.Context) {
	// 	// 注意下面将gin.H参数传入index.tmpl中!也就是使用的是index.tmpl模板
	// 	sign, ts, nonce := c.Query("signature"), c.Query("timestamp"), c.Query("nonce")
	// 	mySign := makeSignature(ts, nonce)
	// 	if mySign == sign {
	// 		echostr := c.Query("echostr")
	// 		logs.Info("ECHO_STR:%s", echostr)
	// 		c.Writer.Write([]byte(echostr))
	// 	}
	// })
	// router.GET("/load", func(c *gin.Context) {
	// 	if i%4 == 0 {
	// 		c.HTML(200, "bangdan.html", gin.H{})
	// 	} else if i%4 == 1 {
	// 		c.HTML(200, "waimai.html", gin.H{})
	// 	} else if i%4 == 2 {
	// 		c.HTML(200, "huanjing.html", gin.H{})
	// 	} else {
	// 		c.HTML(200, "gongzuori.html", gin.H{})
	// 	}
	// 	i = i + 1

	// })
	router.Run(":8002")
}
