# TOS Golang SDK

通用对象存储服务的Go语言SDK


### 例子

        config := &tos.Config{
                Cluster: "default",
                KeyMap: map[string]string{
                        "{bucketName}": "{accessKey}",
                },
        }
        client, err := tos.NewTos(config)
        if err != nil {
                log.Fatal(err)
        }
