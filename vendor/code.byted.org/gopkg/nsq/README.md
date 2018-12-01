# nsq

Go语言，向NSQ发送消息的的基础库。若需要消费消息，可以直接使用官方Go语言包，地址：https://github.com/nsqio/go-nsq


# Example

        import "time"
        import "code.byted.org/gopkg/nsq"

        func main() {
            // 连接nsq实例超时时间
            nsq.ConnectTimeout = 3 * time.Second
            // 读取连接数据的超时时间
            nsq.ReadTimeout = 5 * time.Second
            // 总连接数量的上限
            nsq.LimitConns = 10000
            // 每台机器的连接数量
            nsq.PerHostConns = 10
            // 检查Worker状态的时间间隔
            nsq.Interval = 5
            ch := make(chan *nsq.MsgEntry)
            // 指定nsqd实例的地址列表，**实际使用请替换成`ss_conf/ss/nsq_*.conf`中`nsqd_proxy`的地址列表。Topic申请在[nsq.byted.org](http://nsq.byted.org)**
            servers := []string{"http://127.0.0.1:4101", "http://127.0.0.1:4102", "http://127.0.0.1:4103", "http://127.0.0.1:4104"}
            // multi 表示每个地址启动多少个Worker
            nsqPool := nsq.NewNsqPool(servers, ch, multi)
            nsqPool.Start()
            go nsqPool.Handle()
            // Do Something
        }
