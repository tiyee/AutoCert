# AutoCert

云厂商https证书自动更新机器人(目前只适配阿里云cdn)

## 使用方法

> go run cmd/main.go --domain="xxx.xxx.com" --email=xxx@xxx.com --product=cdn --certonly --dns-access-key-id=xxxxx --dns-access-key-secret=xxxx --cdn-access-key-id=xxxx --cdn-access-key-secret=xxxx

如果只想申请，可以把命令改成:

> go run cmd/apply/main.go --domain="xxx.xxx.com" --email=xxx@xxx.com --product=cdn --certonly --dns-access-key-id=xxxxx --dns-access-key-secret=xxxx --cdn-access-key-id=xxxx --cdn-access-key-secret=xxxx


## 参数说明

| 名称       | 类型     | 说明              |
| ---------- | -------- | ----------------- |
| domain       | `string` | 需要部署证书的域名 |
| email | `string` | 申请制(即你)的邮箱地址    |
| product     | `string` | 产品，目前固定cdn |
| certonly     | `boolean` | 是否新申请 |
| renew     | `boolean` | 是否是更新域名 |
| dns-access-key-id     | `string` | 阿里云dns的AK(主要是申请证书的时候需要添加一条txt记录来验证域名所有权) |
| dns-access-key-secret    | `string` | 阿里云dns的SK |
| cdn-access-key-id     | `string` | 阿里云cdn的AK |
| cdn-access-key-secret     | `string` | 阿里云cdn的SK |

## 说明

dns的秘钥主要是申请证书的时候，let's encrypt需要验证所有权，会写入一条txt解析记录。因此秘钥需要相关的权限

cdn的秘钥是用来查询和部署证书的，也需要相关权限。

dns和cdn可以是相同的秘钥，只需要同时拥有对应权限即可。

申请的证书是RSA2048，目前没有加入配置项，后面可能会加入。

代码是开源的，可自行增加修改。也可以提issue我处理。

## github action 自动运行

直接参考本项目的设置，为了安全，可以把信息放到secret key里
