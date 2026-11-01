# AutoCert

自动申请、续期并部署 HTTPS 证书，使用 Let's Encrypt DNS-01 验证，支持普通域名和泛域名。

## 选择平台

DNS 验证平台和证书部署平台可以不同，也可以使用不同账号的凭据。

| 平台参数 | DNS 验证服务 | 证书部署结果 |
| --- | --- | --- |
| `aliyun` | 阿里云云解析 DNS | 更新阿里云 CDN 域名证书 |
| `tencent` | 腾讯云 DNSPod | **仅上传到腾讯云 SSL 证书库，不绑定 CDN** |
| `esa` | 阿里云 ESA 的 NS 托管 DNS | 配置 ESA 站点证书 |
| `teo` | 腾讯云 EdgeOne 的 NS 接入 DNS | 上传到腾讯云 SSL 后，绑定指定 EdgeOne 加速域名 |

常见组合：

| 使用场景 | `DNS_PLATFORM` | `CDN_PLATFORM` |
| --- | --- | --- |
| 阿里云解析 + 阿里云 CDN | `aliyun` | `aliyun` |
| DNSPod 解析 + 阿里云 CDN | `tencent` | `aliyun` |
| ESA NS 托管 + ESA 证书 | `esa` | `esa` |
| EdgeOne NS 接入 + EdgeOne 证书 | `teo` | `teo` |
| 阿里云解析 + EdgeOne CNAME 接入 | `aliyun` | `teo` |
| DNSPod 解析 + ESA CNAME 接入 | `tencent` | `esa` |

> 主入口的 `--certonly` 表示“申请并部署”，不是仅申请。首次部署使用该参数，后续自动续期时移除它。

## 部署前准备

1. 确认域名的权威 DNS 已生效，部署机器能访问云 API、Let's Encrypt 和公网 DNS。
2. 在云平台完成 CDN 域名、ESA 站点或 EdgeOne 站点的接入及激活。使用 EdgeOne 时，目标加速域名也必须已经存在并完成归属权验证。
3. 准备 DNS 和证书部署两组 AK/SK，并按下文授予权限。两组可以相同，但必须同时具备对应权限。
4. 若使用 STS，准备对应的临时 AK、临时 SK 和 Token，确保它们在本次运行期间有效。

ESA/EdgeOne **仅在 NS 托管接入时用于 DNS 验证**。CNAME 接入、EdgeOne DNSPod 托管接入，应将 `DNS_PLATFORM` 设置为实际的权威 DNS 服务商，证书部署仍可使用 `esa` 或 `teo`。这里的验证是证书申请的 DNS-01 验证，不会替你创建站点、配置源站或完成云平台的站点归属权验证。

## 服务器部署

以下示例使用 Bash，程序、配置与脚本放在同一目录。

### 1. 获取并编译

安装 [go.mod](go.mod) 声明的 Go 工具链版本，然后执行：

```sh
git clone https://github.com/tiyee/AutoCert.git
cd AutoCert
go mod download all
go build -o autocert ./cmd
./autocert --help
```

### 2. 配置域名与凭据

在程序目录创建 `.env`，替换下面的示例值：

```sh
DOMAIN='www.example.com'
EMAIL='admin@example.com'

DNS_PLATFORM='aliyun'
CDN_PLATFORM='aliyun'

DNS_ACCESS_KEY_ID='YOUR_DNS_ACCESS_KEY_ID'
DNS_ACCESS_KEY_SECRET='YOUR_DNS_ACCESS_KEY_SECRET'
CDN_ACCESS_KEY_ID='YOUR_CDN_ACCESS_KEY_ID'
CDN_ACCESS_KEY_SECRET='YOUR_CDN_ACCESS_KEY_SECRET'

# 使用长期 AK/SK 时留空；腾讯云这里填写 SessionToken。
DNS_SECURITY_TOKEN=''
CDN_SECURITY_TOKEN=''

# 仅 ESA / EdgeOne 使用。留空时按域名自动匹配站点。
DNS_SITE_ID=''
CDN_SITE_ID=''

# 仅 ESA / EdgeOne 使用。ESA 默认 cn-hangzhou。
DNS_REGION=''
CDN_REGION=''
```

保护配置文件，不要提交到代码仓库或复制到日志中：

```sh
chmod 600 .env
```

`.env` 由下面的运行脚本读取；程序本身从命令行获取配置，不会自动加载 `.env` 或 YAML 文件。

### 3. 创建运行脚本

在同一目录保存 `run.sh`：

```bash
#!/usr/bin/env bash
set -euo pipefail
cd -- "$(dirname -- "${BASH_SOURCE[0]}")"
source ./.env

exec ./autocert \
  --domain="${DOMAIN:?请配置 DOMAIN}" \
  --email="${EMAIL:?请配置 EMAIL}" \
  --dns-platform="${DNS_PLATFORM:-aliyun}" \
  --cdn-platform="${CDN_PLATFORM:-aliyun}" \
  --dns-access-key-id="${DNS_ACCESS_KEY_ID:?请配置 DNS_ACCESS_KEY_ID}" \
  --dns-access-key-secret="${DNS_ACCESS_KEY_SECRET:?请配置 DNS_ACCESS_KEY_SECRET}" \
  --cdn-access-key-id="${CDN_ACCESS_KEY_ID:?请配置 CDN_ACCESS_KEY_ID}" \
  --cdn-access-key-secret="${CDN_ACCESS_KEY_SECRET:?请配置 CDN_ACCESS_KEY_SECRET}" \
  --dns-security-token="${DNS_SECURITY_TOKEN:-}" \
  --cdn-security-token="${CDN_SECURITY_TOKEN:-}" \
  --dns-site-id="${DNS_SITE_ID:-}" \
  --cdn-site-id="${CDN_SITE_ID:-}" \
  --dns-region="${DNS_REGION:-}" \
  --cdn-region="${CDN_REGION:-}" \
  "$@"
```

```sh
chmod 700 run.sh
```

### 4. 首次申请并部署

确认配置指向预期账号、域名和平台后执行：

```sh
./run.sh --certonly
```

此命令会写入 DNS 验证记录、申请真实证书，并向配置的平台上传或部署证书。ESA/EdgeOne 的成功提示表示云 API 已接受部署，节点生效可能需要时间，请在控制台和实际 HTTPS 访问中确认结果。`CDN_PLATFORM=tencent` 的成功提示仅代表上传完成。

### 5. 设置自动续期

后续执行时**移除 `--certonly`**：

```sh
./run.sh
```

程序查询已有证书，在到期前 10 天重新申请并部署；不需要续期时正常退出。找不到已有证书会报错，不会自动执行首次签发。

可用 `crontab -e` 设置每天检查一次，例如：

```cron
0 3 * * * /opt/autocert/run.sh >> /opt/autocert/renew.log 2>&1
```

请替换为实际安装路径。示例按服务器时区在每天 03:00 运行，定时任务用户需要读取配置和执行程序的权限。日志应保存在受限目录中，并定期查看失败记录。部署多个域名时，为每个域名准备独立的配置目录和定时任务。

## ESA 与 EdgeOne 配置

### 站点 ID

默认按域名自动匹配站点；同名站点无法唯一确定时，需要显式指定 ID。DNS 与部署的 ID 分开填写，不互相回退：

| 配置项 | ESA | EdgeOne |
| --- | --- | --- |
| `DNS_SITE_ID` | 托管验证记录的数字站点 ID | 托管验证记录的 `zone-...` ID |
| `CDN_SITE_ID` | 部署证书的数字站点 ID | 部署证书的 `zone-...` ID |

站点 ID 可从云控制台的站点信息中获取。指定 ID 后仍会查询站点列表、校验域名匹配，因此不能省略站点查询权限。若为 DNS 验证配置了 CNAME 委派，DNS 站点 ID 应指向最终验证记录所在的站点。

### 区域

ESA 默认使用 `cn-hangzhou`。使用国际区域时，在配置文件中设置相应一侧：

```sh
DNS_REGION='ap-southeast-1'
CDN_REGION='ap-southeast-1'
```

如果只有部署使用 ESA，只需设置 `CDN_REGION`。EdgeOne 使用国内 API 端点，区域通常可留空。

### 续期与部署结果

- ESA 查询匹配域名的站点证书，使用实际证书到期时间判断续期；更新对应的 AutoCert 证书，不自动删除其他证书。
- EdgeOne 查询目标加速域名已部署的证书；同时绑定多张证书时，按最早到期时间判断。证书仍在部署中时会报错，请等待后再检查。
- EdgeOne 上传成功、绑定失败时会返回错误和已上传证书 ID。请先检查控制台及权限，不要把上传成功视为域名证书已更新。
- 程序不自动清理腾讯云 SSL 证书库中的旧证书，请按需在控制台维护证书配额。

## 云账号权限

按所用功能授予权限；DNS 凭据与部署凭据可以分别授权到相应域名、站点或资源。下表列出程序使用的操作，具体资源范围以云平台权限控制要求为准。

| 平台与用途 | 需要允许的操作 |
| --- | --- |
| 阿里云 DNS 验证 | `alidns:AddDomainRecord`、`alidns:DeleteDomainRecord` |
| 阿里云 CDN 部署/查询 | `cdn:DescribeDomainCertificateInfo`、`cdn:SetCdnDomainSSLCertificate` |
| 腾讯云 DNSPod 验证 | `dnspod:CreateRecord`、`dnspod:DeleteRecord` |
| 腾讯云 SSL 上传/查询 | `ssl:UploadCertificate`、`ssl:DescribeCertificates` |
| ESA DNS 验证 | `esa:ListSites`、`esa:CreateRecord`、`esa:DeleteRecord` |
| ESA 部署/查询 | `esa:ListSites`、`esa:ListCertificatesByRecord`、`esa:GetCertificate`、`esa:SetCertificate` |
| EdgeOne DNS 验证 | `teo:DescribeZones`、`teo:CreateDnsRecord`、`teo:DeleteDnsRecords` |
| EdgeOne 部署/查询 | `teo:DescribeZones`、`teo:DescribeAccelerationDomains`、`teo:ModifyHostsCertificate`；部署还需 `ssl:UploadCertificate` |

使用 STS 时，角色权限和会话策略也必须允许这些操作。Token 不能替代 AK/SK，程序不会自动 AssumeRole 或刷新临时凭据；定时任务运行前需由外部流程更新完整的临时凭据。

## GitHub Actions 部署

仓库提供 [.github/workflows/AutoCert.yml](.github/workflows/AutoCert.yml)，支持定时检查和手动触发。

1. Fork 仓库，在自己的仓库中启用 Actions，并检查定时工作流是否已启用。
2. 打开 `Settings → Secrets and variables → Actions`，配置下面的凭据。
3. 编辑工作流，只保留自己需要的域名步骤，替换对应域名变量及平台参数。仓库自带的多个域名和专用账号配置是部署示例，不能直接作为自己的配置使用。
4. 先按“首次申请并部署”完成签发；工作流默认使用 `--renew`，无法替没有证书的域名执行首次部署。
5. 在 Actions 页面手动运行 `AutoCert`，确认每个域名的结果后，再依赖定时任务。

通用凭据使用以下 Repository Secrets：

| Secret | 内容 | 是否必填 |
| --- | --- | --- |
| `EMAIL` | 申请者邮箱 | 是 |
| `DNSACCESSKEYID` | DNS AK | 是 |
| `DNSACCESSKEYSECRET` | DNS SK | 是 |
| `CDNACCESSKEYID` | 部署 AK | 是 |
| `CDNACCESSKEYSECRET` | 部署 SK | 是 |
| `DNSSECURITYTOKEN` | DNS STS Token | 使用 DNS 临时凭据时填写 |
| `CDNSECURITYTOKEN` | 部署 STS Token | 使用部署临时凭据时填写 |

域名填写在工作流所引用的 Repository Variables 中，例如 `DOMAINTOOLS`。可以保留该变量名并填入自己的域名，也可以同时修改变量名和工作流引用。泛域名填写 `*.example.com`，不要在变量值中额外加引号。

使用专用账号的步骤，Secret 名称以该步骤的 `env` 为准。例如 `AVAR_DNS_AK` / `AVAR_DNS_SK` 对应 `AVAR_DNS_SECURITY_TOKEN`，`SEED_CDN_AK` / `SEED_CDN_SK` 对应 `SEED_CDN_SECURITY_TOKEN`。未使用的步骤应删除，避免缺少配置导致任务失败。

默认平台为阿里云 DNS + 阿里云 CDN。使用 ESA/EdgeOne 时，需要在对应步骤的命令中添加 `--dns-platform=esa`、`--cdn-platform=teo` 等实际组合；也可添加 `--dns-site-id`、`--cdn-site-id` 和区域参数。凭据继续通过 Secrets 注入，站点 ID 和区域可放在 Repository Variables 中。

当前定时表达式为 `30 5 */5 * *`：在每月的 1、6、11、16、21、26、31 日（当月存在该日期时），UTC 05:30、北京时间 13:30 运行，并非严格每隔 120 小时。每个域名在独立步骤中运行，日志会先显示域名和配置变量名。申请或部署失败时，直接打印程序错误和该域名的退出码，然后继续处理其余域名；不会再执行统一的失败汇总或 `exit 1`。

查看 Actions 中对应域名的步骤即可找到完整错误。域名处理失败不会使整个工作流失败，因此工作流显示成功不代表所有域名都处理成功，请检查各域名日志。依赖下载或构建失败仍会使工作流失败。

STS Token 会过期，不宜将一次获取的短期 Token 当作长期定时任务凭据；使用 STS 的工作流需要在每次运行前获取有效凭据，现有工作流不会自动刷新。

## 命令行参数

既有命令继续可用，支持 `-platform=aliyun` 和 `--platform=aliyun` 两种写法。

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `--domain` | 无 | 必填，申请/部署的域名；泛域名需加引号 |
| `--email` | 无 | 必填，申请者邮箱；查询入口也要求提供 |
| `--platform` | `aliyun` | DNS 与部署的默认平台：`aliyun`、`tencent`、`esa`、`teo` |
| `--dns-platform` | 跟随 `platform` | 单独指定 DNS 验证平台 |
| `--cdn-platform` | 跟随 `platform` | 单独指定证书部署/上传平台 |
| `--dns-access-key-id` / `--dns-access-key-secret` | 无 | DNS AK/SK |
| `--cdn-access-key-id` / `--cdn-access-key-secret` | 无 | 部署 AK/SK |
| `--dns-security-token` | 空 | DNS STS Token；腾讯云为 SessionToken |
| `--cdn-security-token` | 空 | 部署 STS Token；腾讯云为 SessionToken |
| `--dns-site-id` / `--cdn-site-id` | 自动匹配 | 仅 ESA/EdgeOne 使用，各自独立 |
| `--dns-region` / `--cdn-region` | ESA 为 `cn-hangzhou` | 仅 ESA/EdgeOne 使用，各自独立 |
| `--certonly` | `false` | 主入口申请并部署；定时续期时不要添加 |
| `--renew` | 兼容参数 | 主入口未设置 `certonly` 时执行续期；`--renew=false` 不会禁用续期 |
| `--product` | 空 | 兼容旧命令，当前不用于选择云产品；使用平台参数选择部署目标 |
| `--version` | `false` | 输出版本并退出 |
| `--help` | — | 查看完整参数帮助 |

泛域名申请会同时包含去掉 `*.` 的域名，例如 `*.sub.example.com` 同时包含 `sub.example.com`。当前证书密钥为 RSA 2048。

如需使用独立入口：

| 入口 | 用途 | 凭据要求 |
| --- | --- | --- |
| `go run cmd/main.go` | 申请部署或自动续期 | DNS 和部署两组凭据 |
| `go run cmd/apply/main.go` | 仅申请证书 | DNS 凭据 |
| `go run cmd/deploy/main.go` | 仅查询证书状态，**不会执行部署** | 部署凭据 |

三个入口均需传入域名和邮箱。仅申请入口会输出包含私钥的申请结果，应使用受限文件保存（例如先执行 `umask 077`），不要写入公开日志或 Actions 日志。

## 常见问题

| 现象 | 处理方法 |
| --- | --- |
| `no certs found` | 首次运行使用 `--certonly`；已有证书则检查平台、账号及域名绑定情况 |
| `SignatureDoesNotMatch` | 请求签名不匹配；检查报错一侧 AK/SK 是否配套、是否含多余空格或换行；临时凭据还需配套 Token。DNS 创建记录报错时检查 DNS 凭据，不要改动无关的部署凭据 |
| 权限不足或 Token 失效 | 检查对应一侧 AK/SK/Token、角色会话策略和上述操作权限；STS 要整组更新 |
| 找不到站点或站点不匹配 | 检查账号、ESA 区域、站点 ID；有 DNS CNAME 委派时检查最终验证记录所在站点 |
| 多个同名站点无法确定 | 分别配置 `DNS_SITE_ID`、`CDN_SITE_ID` |
| 站点不是 NS 托管 | 将 DNS 平台改为实际权威 DNS 服务商，保留原部署平台 |
| TEO 目标不存在、待归属权验证或已停用 | 先在 EdgeOne 控制台完成接入、验证并启用目标加速域名 |
| DNS 验证超时 | 检查 NS 是否生效、TXT 能否公开查询、CNAME 委派及 CAA 限制；ESA/EdgeOne 验证最多等待 20 分钟 |
| 上传成功但域名仍是旧证书 | 先确认是否选择了仅上传的 `tencent`；EdgeOne 检查绑定结果，ESA/EdgeOne 等待节点生效后再检查实际 HTTPS |
| 提示证书仍在部署中 | 等待云端部署完成后重试，不要反复执行首次申请 |
| 某个域名处理失败 | 打开对应域名步骤，查看原始错误及退出码；其他域名仍会继续处理，无需全部重新签发 |
| 工作流成功但个别域名未更新 | 域名错误只打印日志、不会使工作流失败；请检查各域名步骤 |

程序出错返回非零退出码；证书无需续期时返回成功。排查问题时只提供脱敏错误码，不要公开 AK、SK、Token 或私钥。
