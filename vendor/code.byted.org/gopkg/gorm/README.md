# GORM

Mirror of github.com/jinzhu/gorm

## Context功能
目前gorm已经支持context功能;

将上层的context传递下去, 能够帮助底层的基础库获得一些有用的信息;

使用方式如下:

```
db = db.Context(ctx)
```