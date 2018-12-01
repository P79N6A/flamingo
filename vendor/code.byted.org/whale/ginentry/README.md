# GinEntry
为所有基于Gin的HTTP服务提供接入Whale反爬取服务的中间件。 
在LB和client都设置了严格的10ms超时，异常近最大可能不影响宿主服务。  
   
建议将Middleware加在所有的原有业务逻辑的middleware之前，保证攻击者的请求在被whale拦截之后不会影响业务逻辑，造成数据不一致。  
所有的拦截或者其它决策来自于在whale.byted.org上的规则与策略的配置。  
  
当前支持决策：  
0. PASS 通过。
1. BLOCK 返回code 403。请求不会到达业务方的handlers。  
2. MOCK 重定向到配置的静态链接，请求不会到达业务方的handlers。用于迷惑攻击者。  
3. CUSTOM 识别为可疑的攻击者后，透传给业务方的handlers，由业务方在平台的配置的数据做出自定义的决策，如返回假数据等。  
4. CAPTCHA 人机识别。通过图片验证、滑块等手段做人机识别。当前暂时未支持。  
  
# demo
```go
import (
    "code.byted.org/whale/ginentry"
    "github.com/gin-gonic/gin"
)

func main() {
    g := gin.Default()

    //参数为不过Whale反爬取服务的白名单path，使用前缀匹配当前请求的path，命中则不过反爬取服务
    anticConf := ginentry.NewAntiCrawlConfig([]string{"/not/to/whale"})
    g.Use(ginentry.AntiCrawl(anticConf))

    g.GET("/not/to/whale", func(c *gin.Context) {
        c.String(200, "What a shame")
    })

    g.Run(":8080")
}
```
  
如果需要配置黑名单（只有在黑名单出现的url前缀匹配的请求才发送反爬服务），则配置如下：  
```go
import (
    "code.byted.org/whale/ginentry"
    "github.com/gin-gonic/gin"
)

func main() {
    g := gin.Default()

    //参数为不过Whale反爬取服务的白名单path，使用前缀匹配当前请求的path，命中则不过反爬取服务
    anticConf := ginentry.NewAntiCrawlConfigWithBlackPaths([]string{"/black/has/white"}, []string{"/black"})
    g.Use(ginentry.AntiCrawl(anticConf))

    g.GET("/black/has/white", func(c *gin.Context) {
        c.String(200, "What a shame")
    })
    g.GET("/black/to/check", func(c *gin.Context) {
        c.String(200, "nice")
    })

    g.Run(":8080")
}
```
  
__当同时配置了黑名单和白名单时，只有命中了黑名单且没有命中白名单的请求会去检查反爬服务。__