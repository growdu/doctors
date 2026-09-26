# 密钥管理最佳实践（Doctoes 平台）

> 适用范围：陪诊师平台所有 Go 服务 + 3 中间件（PostgreSQL / Redis / Kafka）的**生产环境**
> 密钥（DB 密码、Redis 密码、JWT 签名密钥、Kafka SASL、OTel collector token、第三方 API
> key 等）。本文档是仓库内对 ops 团队的**唯一**密钥管理规章。

---

## 目录

1. [核心原则](#1-核心原则)
2. [生产环境密钥管理：4 种方案对比](#2-生产环境密钥管理-4-种方案对比)
3. [密钥轮转策略](#3-密钥轮转策略)
4. [Secret 泄漏应急 SOP](#4-secret-泄漏应急-sop)
5. [CI / CD 密钥注入](#5-管-cd-密钥注入)
6. [dev / prod 分离](#6-dev-prod-分离)
7. [常见反模式](#7-常见反模式)
8. [附录：密钥检查清单](#8-附录密钥检查清单)

---

## 1. 核心原则

- **绝不提交**：真实密钥**永远不进入 git 仓库**（含二进制、tag、PR、issue 内容）。
- **绝不打印**：生产密钥**永远不写入日志**（zap field 名为 `secret` / `password` /
  `token` 的字段一律 redact；recovery / gin recovery 中也过滤）。
- **最小权限**：每个服务**独立密钥**（auth / order / ... 一对一，不共用），授权范围
  仅限本服务能接触的中间件。
- **可轮转**：所有密钥在生产都**可热替换**——服务启动时通过 secret store 拉取，
  `SIGHUP` 或文件 mount 更新可触发 reload，无须改镜像。
- **可审计**：每次密钥访问（拉取 / 轮转 / 失败）都有结构化日志 + 集中审计（Vault
  audit log / 云 KMS 操作日志）。
- **零信任**：开发者本地**拿不到任何生产密钥**——本地连 dev 数据库用 `.env.dev`
  模板，绝不拉真库。

---

## 2. 生产环境密钥管理：4 种方案对比

下列方案均满足"密钥不入仓库 + 可轮转 + 可审计"。根据团队规模 / 云形态挑选。

### 方案 1：Docker Secrets（推荐小团队 / 单主机）

- **原理**：Docker 19.03+ 内置 secret 子系统；密钥以 tmpfs 文件形式 mount 到
  `/run/secrets/<name>`，镜像内 read-only；swarm / compose v3.8+ 支持。
- **使用**：
  ```yaml
  # docker-compose.prod.yml
  services:
    auth-service:
      image: doctors/auth:latest
      secrets:
        - doctors_jwt_secret
        - doctors_db_password
  secrets:
    doctors_jwt_secret:
      file: ./secrets/jwt_secret.txt   # 生产由 init container / Ansible 写入
    doctors_db_password:
      external: true                   # 来自 docker secret 子系统外部存储
  ```
  服务内读取：
  ```go
  secret, _ := os.ReadFile("/run/secrets/doctors_jwt_secret")
  cfg.Auth.JWTSecret = strings.TrimSpace(string(secret))
  ```
- **优点**：零依赖（纯 Docker）；mount tmpfs 不落盘；与 compose 生态天然兼容；
  小团队 / 单主机上线快。
- **缺点**：缺少自动轮转 / 审计 / 跨主机同步；secret 一旦 mount 不更新（须重启）。
- **适用**：

| 团队规模 | 主机数 | 是否推荐 |
| -------- | ------ | -------- |
| 1–5 人   | 1 台   | ✅       |
| 5–20 人  | 2–5 台 | ⚠ 凑合（建议升级到 K8s 或 Vault）|
| ≥ 20 人  | ≥ 5 台 | ❌ 不推荐 |

### 方案 2：Kubernetes Secrets + sealed-secrets（推荐 K8s 用户）

- **原理**：`Secret` 是 K8s 内置对象，base64 编码存入 etcd；可通过 `SealedSecret`
  （[bitnami-labs/sealed-secrets](https://github.com/bitnami-labs/sealed-secrets)）
  进行**公钥加密**——仓库内可提交密文，由 controller 在集群内私钥解密为明文 Secret。
- **使用**：
  ```bash
  # 一次性：安装 controller 并导出公钥
  helm install sealed-secrets sealed-secrets/sealed-secrets
  kubeseal --fetch-cert > pubkey.pem
  # 加密：
  kubectl create secret generic auth-jwt --dry-run=client \
    --from-literal=jwt-secret=<random> -o yaml | \
    kubeseal --cert pubkey.pem > sealed-auth-jwt.yaml
  # 提交 sealed-auth-jwt.yaml 到仓库（明文 jwt-secret 不会被 push）
  git add sealed-auth-jwt.yaml
  ```
  K8s Deployment 引用：
  ```yaml
  envFrom:
    - secretRef:
        name: auth-jwt  # controller 已还原为 Secret
  ```
- **优点**：与 GitOps 兼容（密文可入库）；controller 私钥可由 KMS 托管；自动注入；
  无外部依赖。
- **缺点**：轮转须触发 controller 重加密 + Deployment restart；缺少细粒度动态
  secret；审计弱（依赖 K8s audit log）。
- **适用**：

| 场景                         | 是否推荐 |
| ---------------------------- | -------- |
| 已在 K8s 上跑生产            | ✅ 强烈推荐 |
| 多环境（staging/prod）同 GitOps | ✅ 强烈推荐 |
| 单机 / 非 K8s                | ❌（改用方案 1 / 3）|

### 方案 3：HashiCorp Vault（推荐多服务多租户）

- **原理**：Vault 是 secret 中心化服务；支持 KV v2（静态密钥）、动态密钥（每次请求
  生成新 DB 用户名 / 密码）、PKI、Transit（加解密）、租约（lease）+ 自动 revoke。
  服务启动时通过 `VAULT_TOKEN` 拉取，或更安全地走 [Kubernetes auth method](https://www.vaultproject.io/docs/auth/kubernetes)。
- **使用**（KV v2 示例）：
  ```bash
  # 一次性：启用 kv v2 + 写密钥
  vault secrets enable -path=doctors kv-v2
  vault kv put doctors/auth/jwt value=$(openssl rand -hex 32)
  vault kv put doctors/db/postgres password=$(openssl rand -hex 16)

  # 服务内：用 vault agent 边车 / 应用 SDK 拉取
  vault kv get -format=json doctors/auth/jwt | jq '.data.data.value'
  ```
  Sidecar 注入（推荐）：
  ```hcl
  # /etc/vault-agent.hcl
  auto_auth {
    method "kubernetes" { ... }
  }
  template {
    destination = "/etc/secrets/auth-jwt"
    contents     = "{{ with secret "doctors/auth/jwt" }}{{ .Data.data.value }}{{ end }}"
  }
  ```
- **优点**：动态 DB 用户（每次轮转不需重启服务）；lease + 自动 revoke；完整 audit
  log；支持多租户 / 多环境 namespace；插件丰富（database / pki / aws / gcp）。
- **缺点**：运维复杂度高；高可用须 Raft / Consul 后端；新人学习曲线陡；Vault
  自身 HA 的密钥管理又是新问题（unseal key 拆分）。
- **适用**：

| 场景                                  | 是否推荐 |
| ------------------------------------- | -------- |
| ≥ 10 服务 / ≥ 3 租户                  | ✅ 强烈推荐 |
| 必须动态 DB 用户 / 短时凭证           | ✅ 唯一推荐 |
| 已在用 Terraform / Consul / Nomad     | ✅ 强烈推荐 |
| 小团队 / 1–2 应用                     | ⚠ 杀鸡用牛刀（改方案 1 / 2）|

### 方案 4：云原生 secret manager（AWS / GCP / Azure）

- **原理**：把密钥托管给云厂商 Key Management Service：
  - **AWS Secrets Manager**：按 secret 收费 $0.40/月 + API 调用 $0.05/万次；
    支持 Lambda 轮转。
  - **AWS SSM Parameter Store（SecureString）**：与 KMS 集成；更便宜；批量查询方便。
  - **GCP Secret Manager**：类似 AWS；版本化；IAM 细粒度。
  - **Azure Key Vault**：集成 Entra ID；HSM 支持。
- **使用**（AWS Secrets Manager）：
  ```go
  // Go SDK
  client := secretsmanager.NewFromConfig(cfg)
  out, _ := client.GetSecretValue(context.Background(),
      &secretsmanager.GetSecretValueInput{
          SecretId: aws.String("doctors/auth/jwt"),
      })
  secret := *out.SecretString
  ```
  配合 IRSA（IAM Roles for Service Accounts）/ GCP Workload Identity：**Pod 不再需要
  任何 access key**，由 IAM 自动交换。
- **优点**：零运维；99.9% SLA；KMS 自动加密；审计日志接入 CloudTrail / Cloud Audit
  Logs；合规友好（PCI DSS / HIPAA / SOC2）。
- **缺点**：跨云不通用（AWS secret 不能用于 GCP）；单区域；成本随 secret 数量 +
  API 调用量上升；网络抖动时本地缓存必须做好。
- **适用**：

| 场景                | 推荐产品                       |
| ------------------- | ------------------------------ |
| 全栈 AWS            | Secrets Manager + IRSA         |
| 全栈 GCP            | GCP Secret Manager + Workload Identity |
| 全栈 Azure          | Azure Key Vault + Managed Identity      |
| 多云混合            | ❌ 用方案 3（Vault）|

### 4 种方案对比表

| 维度           | Docker Secrets | K8s + Sealed-Secrets | HashiCorp Vault    | 云 Secret Manager |
| -------------- | :------------: | :------------------: | :----------------: | :---------------: |
| 学习曲线       | 低             | 中                   | **高**             | 低                |
| 运维成本       | 极低           | 低                   | **高**             | 极低（SaaS 化）   |
| 动态密钥       | ❌             | ❌                    | ✅                  | ✅（Lambda）      |
| 自动轮转       | 手动           | 手动                 | ✅                  | ✅                 |
| 审计日志       | 弱             | K8s audit            | **完整**           | 云审计日志        |
| 多租户         | ❌             | ❌                    | ✅                  | ✅                 |
| 跨云           | ✅             | ✅                    | ✅                  | ❌                |
| 单 secret 月成本 | 0             | 0                    | $0（社区版）       | $0.40 起          |
| 推荐规模       | 1–5 服务       | 5–20 服务            | 10+ 服务 / 多租户 | 全栈单一云       |
| 推荐组合       | 单主机 demo    | GitOps / K8s 团队    | 微服务平台        | 云原生            |

**Doctoes 平台当前阶段推荐**：

- dev / staging → Docker Secrets 或直接 `.env.dev` + docker-compose
- prod（≤ 5 服务）→ **K8s + Sealed-Secrets**（既保留 GitOps 又不引入 Vault 复杂度）
- prod（≥ 10 服务 / 多租户） → **Vault 动态密钥**

---

## 3. 密钥轮转策略

> 原则：**有期限 + 有审计 + 有回滚**。绝不无限期使用任何密钥。

### 3.1 轮转频率

| 密钥类型             | 轮转周期 | 轮转触发         | 灰度策略                          |
| -------------------- | -------- | ---------------- | --------------------------------- |
| **JWT_SECRET**       | 90 天    | 定时 + 事件      | 双密钥并存 7 天（v1 / v2 并行校验）|
| **DB 密码**          | 180 天   | 定时 + 离职       | 新老密码并行 24h（应用双连接 fallback）|
| **Kafka SASL**       | 365 天   | 定时             | 滚动重启（按消费者组顺序）        |
| **Redis 密码**       | 365 天   | 定时             | 重启 + sentinel failover          |
| **第三方 API key**   | 90 天    | 每次涉及人就轮转 | 立即生效（调用方实现 401 重试）   |
| **TLS 私钥**         | 90 天    | 定时             | 双证书并存 + ACME 自动续签        |

> **关键约定**：JWT_SECRET 轮转时，**必须支持新老 token 一段窗口期共存**。
> 详见 `services/auth/internal/jwt/*` 中的 KeySet 切换逻辑（计划 §34+）。

### 3.2 轮转 SOP（以 JWT_SECRET 为例）

```bash
# 1. 在 secret store 生成新密钥（不放任何明文日志）
NEW_SECRET=$(openssl rand -hex 32)
vault kv put doctors/auth/jwt value="$NEW_SECRET" version=2

# 2. 灰度：sealed-secrets 写一个新密钥，旧密钥保留
#    services/auth 通过 KEY_SET=current,previous 双密钥验证 token
#    老 token 7 天后过期，previous 自动撤销

# 3. 触发服务 reload（Vault agent 模板更新 → 文件更新 → SIGHUP）
kubectl rollout restart deployment/auth-service

# 4. 验证：登录新发 token + 老 token 均能通过
curl -fsS http://auth:8080/healthz
# （用老 token 调受保护 API，确认仍可鉴权）
```

### 3.3 轮转监控

Prometheus 指标（部署后接入）：

```promql
# secret 拉取失败率（来自 secret-client 库）
rate(secret_fetch_failures_total[5m]) > 0.01

# 密钥版本滞后（服务使用的密钥版本 < 集群中最新版本超过 24h）
time() - secret_last_refreshed_timestamp_seconds > 86400
```

---

## 4. Secret 泄漏应急 SOP

> **窗口期**：从发现泄漏到完成密钥全量轮转，**目标 ≤ 4 小时**。泄漏后无预案等于
> 系统全裸。

### 4.1 检测（Detection）

触发源包括但不限于：

- [GitHub Secret Scanning](https://docs.github.com/en/code-security/secret-scanning)
  自动告警（push 触发）
- [TruffleHog](https://github.com/trufflesecurity/trufflehog) 定期扫描历史 commit
- [gitleaks](https://github.com/gitleaks/gitleaks) 在 CI 中预提交扫描
- Vault 异常 audit log（短时大量 `kv get`）
- 云厂商异常 KMS 操作告警

### 4.2 隔离（Containment，0–15 分钟）

1. **立刻封口**：确认泄漏范围（commit SHA / log 行号 / issue 截图）。
2. **从 git 历史移除**（仅在极端情况）：
   ```bash
   # 用 BFG Repo-Cleaner 重写 history（**慎用** —— 协调全团队同步 clone）
   bfg --delete-files id_rsa
   git reflog expire --expire=now --all
   git gc --prune=now --aggressive
   # force push 并通知所有协作者重新 clone
   ```
   > ⚠ **首选方案是直接轮转密钥**，而不是清理 git history（git history 一旦被
   > fork 就在别人手里了，删了也白搭）。
3. **撤销泄漏的凭证**：
   - JWT_SECRET → 触发轮转 + 强制所有用户下线重新登录
   - DB 密码 → 立即更换 + 监控连接来源 IP
   - 云 access key → IAM disable / delete + 检查 CloudTrail 是否有异常调用
4. **临时撤权**（如果攻击者可能已活跃）：
   ```bash
   # 把对应 pod 临时踢出集群
   kubectl cordon auth-service
   # 切断网络
   kubectl apply -f emergency-network-policy.yaml
   ```

### 4.3 重新生成（Regenerate，15 分钟 – 2 小时）

1. 由 2 人协作生成新密钥（一人提 PR，一人 review）。
2. 通过 vault / sealed-secret / 云 KMS 写入。
3. 触发滚动重启 / hot reload。
4. 用 **新密钥**生成的新凭证在 staging 跑一遍冒烟。

### 4.4 重启服务（Restart，2 – 4 小时）

1. 按依赖顺序滚动重启：infra (PG/Redis/Kafka) → auth → 用户面服务 → admin。
2. 老凭证全部失效后，验证：
   ```bash
   # 验证所有服务使用新密钥握手成功
   for svc in auth order match message payment review sos user escort wallet admin; do
     curl -fsS http://localhost:${port[$svc]}/healthz
   done
   ```
3. 检查 audit log：无新密钥拉取失败的告警。

### 4.5 复盘（Postmortem，72 小时内）

- 写内部事故报告（**无指责**语言）：发生了什么 / 如何发现 / 影响面 / 为什么能
  进入仓库 / 防护改进。
- 把改进项纳入下个 sprint：例如
  - 加 gitleaks pre-commit hook
  - 加密静态 configmap
  - 给敏感目录加保护分支规则

---

## 5. CI / CD 密钥注入

### 5.1 GitHub Actions

- **secret 存储**：[GitHub Encrypted Secrets](https://docs.github.com/en/actions/security-guides/encrypted-secrets)
  per-repo / per-env；runner 解密注入到 env，**不会在 log 中打印**（除非显式 echo）。
- **使用**：
  ```yaml
  # .github/workflows/prod-deploy.yml
  jobs:
    deploy:
      environment: production        # 受 environment protection rule 保护
      steps:
        - uses: aws-actions/configure-aws-credentials@v4   # OIDC → 无 long-lived key
          with:
            role-to-assume: arn:aws:iam::123:role/doctors-deployer
        - run: kubectl apply -f deploy/
          env:
            KUBECONFIG: ${{ secrets.KUBECONFIG }}
            REGISTRY_TOKEN: ${{ secrets.REGISTRY_TOKEN }}
  ```
  > 推荐 **OIDC + IRSA**：CI runner 用短期 token 换 AWS role，不用任何 long-lived
  > access key。

### 5.2 SOPS（Secrets OPerationS）

[mozilla/SOPS](https://github.com/getsops/sops) = YAML/JSON 上的透明加密层；与
age / KMS / PGP 集成。

- **使用**：
  ```bash
  # 用 KMS 加密（推荐）
  export SOPS_KMS_ARN="arn:aws:kms:ap-east-1:123:key/abcd-..."
  sops --encrypt --kms $SOPS_KMS_ARN \
    --in-place deploy/k8s/secrets.enc.yaml

  # CI 中解密：注入 env 或临时文件
  sops --decrypt deploy/k8s/secrets.enc.yaml | kubectl apply -f -
  ```
- **优点**：
  - 密文可入库（同 Sealed-Secrets 的 GitOps 体验）。
  - 加密粒度按字段（仅 `data.password` 加密，metadata 仍可见）。
  - 与 Terraform / Kustomize 无缝。
- **缺点**：CI 必须能访问 KMS / age 私钥；权限过宽 = 风险。

### 5.3 推荐组合

| 环境               | secret 注入方式                                            |
| ------------------ | ---------------------------------------------------------- |
| 本地 dev           | `.env.dev` 模板（仓库内）                                   |
| CI（linter / test）| GitHub Secrets（短时 token / mock credential）              |
| staging deploy     | K8s Sealed-Secret → Helm values / SOPS                     |
| prod deploy        | Vault agent sidecar（动态）或云 IRSA + External Secrets    |

---

## 6. dev / prod 分离

### 6.1 严格禁止

- ❌ 用生产数据库调试代码
- ❌ 把生产 JWT_SECRET 复制到本地 `.env`
- ❌ 让本地服务指向 `api.doctors.com`
- ❌ 在 prod configmap 里写明文密码（即使是 staging 集群）
- ❌ 让 prod 凭证出现在 PR 截图 / 录屏

### 6.2 强制约束

| 项                | dev / staging                  | prod                              |
| ----------------- | ------------------------------ | --------------------------------- |
| 数据库            | docker-compose `postgres:16`    | RDS / Aurora / 自管高可用集群     |
| Redis             | docker-compose `redis:7`        | ElastiCache / 哨兵集群            |
| Kafka             | docker-compose 单节点 KRaft    | 3 broker + 镜像 + tiered storage  |
| JWT_SECRET        | `dev-secret-change-me`         | openssl rand -hex 32 每服务独立   |
| OTel collector    | 不接 / Jaeger all-in-one       | 专用 OTel collector + Kafka backend|
| secret 注入       | `.env.dev`                     | Vault / Sealed-Secret / IRSA      |
| 日志保留          | 不超过 24h                     | 90d 合规保留                       |
| 访问审计          | 关                             | 全开（K8s audit + Vault audit）    |

### 6.3 实现机制

- **CI 分环境变量读取**：staging 和 prod 用不同 GitHub Environment secret，
  PR 自动 deploy 到 staging，main merge 后**人工 approve**才到 prod。
- **服务端做来源校验**：所有服务启动时校验 `ENV=production` 时**禁止**
  监听 `:8080`，必须挂载证书（如 jaeger / cert-manager）；也禁止 fallback 到
  dev host 名（如 `127.0.0.1`、`localhost`）。
  ```go
  // shared/config/loader.go
  if cfg.ServiceVersion == "dev" && env == "production" {
      return nil, fmt.Errorf("dev config in production env")
  }
  ```
- **kubectl-allowed-repos 限定**：`deploy/` 目录的 PR 只能在指定 team member
  review 后合并，自动化 bot 不允许直接 push。

---

## 7. 常见反模式

| 反模式                                | 风险                          | 正确做法                          |
| ------------------------------------- | ----------------------------- | --------------------------------- |
| 把 JWT 密钥硬编码在 `cfg.Auth.JWTSecret` 默认值 | 一旦代码泄漏 = 全员可签发 token | 默认值必须为 `""`，空时报错退出    |
| 把 `cfg/k8s/secret.yaml` 入库        | 所有人可见生产凭证            | 用 Sealed-Secret / SOPS 加密      |
| 用 git tag / commit message 携带 token | 永久留痕                      | tag 只含版本号，token 走外部通道  |
| 服务启动失败时打印完整 config          | 日志泄漏密码                  | log redact `secret` / `password` 字段 |
| 11 个服务共用一个 JWT_SECRET          | 1 个泄漏 = 11 个失守           | 每服务独立 JWT_SECRET             |
| 轮转密钥后忘了 bump deployment       | 服务仍持有旧密钥（lease 失效） | 轮转强制触发 rollout restart      |
| CI cache 中残留 secret                | 后续 job 复用                 | 清理 `$GITHUB_ENV` / temp file    |
| 在 PR / issue 截图发生产报错          | URL/cookie 暴露                | 截图前 redact；issue 模板提醒     |

---

## 8. 附录：密钥检查清单

部署到任何环境前，逐项 ✅：

- [ ] `.env.prod` / secret manifest 中**无明文生产密钥**
- [ ] 所有 service `JWT_SECRET` **相互独立**，且 ≠ `dev-secret-change-me`
- [ ] DB / Redis / Kafka 密码**通过 secret store 注入**，不在 compose YAML 中明写
- [ ] `git log --all --full-history -- "*.env"` 返回空
- [ ] `gitleaks detect --no-git` 全绿
- [ ] CI workflow 不出现 `${{ secrets.* }}` 的 echo / debug 输出
- [ ] Kubernetes NetworkPolicy 已配置：prod 命名空间不允许外部 IP 直连 PG/Redis
- [ ] OTel collector / 外发 API 的 secret 已挂载证书 / token
- [ ] Prometheus 告警 `SecretFetchFailures > 0` / `SecretVersionStale > 24h` 已配置
- [ ] 团队已演练过 Secret 泄漏应急 SOP（≤ 6 个月 1 次）

---

> 文档维护：本文件随 secret store 选型一起演进；任何 schema / 工具变更都需经
> SRE lead review。联系：`#ops` Slack 频道。
