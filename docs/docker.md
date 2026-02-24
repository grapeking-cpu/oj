# Docker 沙箱隔离与多语言镜像方案

## 一、镜像架构

### 1.1 镜像列表

| 镜像名 | 语言版本 | 基础镜像 | 大小 | 用途 |
|--------|----------|----------|------|------|
| `oj-runner-cpp17` | GCC 13 | alpine:3.18 | ~150MB | C++17 |
| `oj-runner-c11` | GCC 13 | alpine:3.18 | ~150MB | C11 |
| `oj-runner-go122` | Go 1.22 | alpine:3.18 | ~180MB | Go 1.22 |
| `oj-runner-py311` | Python 3.11 | alpine:3.18 | ~200MB | Python 3.11 |
| `oj-runner-java17` | OpenJDK 17 | eclipse-temurin:17-jre | ~250MB | Java 17 |
| `oj-runner-cpp-spj` | GCC 13 | alpine:3.18 | ~150MB | SPJ 判题 |

### 1.2 设计原则

1. **最小化**: 使用 alpine，减少攻击面
2. **语言隔离**: 一语言一镜像，互不影响
3. **只读挂载**: 代码/测试数据只读挂载
4. **禁网**: 禁用网络访问 (除非需要远程评测)
5. **资源限制**: cgroups 限制 CPU/内存/进程数

---

## 二、镜像 Dockerfile 示例

### 2.1 C++17 镜像

```dockerfile
# oj-runner-cpp17/Dockerfile
FROM alpine:3.18

# 安装编译工具
RUN apk add --no-cache \
    g++=13.2.1-r3 \
    libc-dev=0.7.2-r3 \
    make=4.4.1-r1

# 创建工作用户 (非 root)
RUN adduser -D -s /bin/sh runner

# 设置工作目录
WORKDIR /home/runner

# 切换用户
USER runner

# 默认命令
CMD ["sh"]
```

### 2.2 Python 镜像

```dockerfile
# oj-runner-py311/Dockerfile
FROM python:3.11-slim

# 安装必要包
RUN pip install --no-cache-dir --break-system-packages \
    numpy==1.26.0 \
    scipy==1.11.0

# 创建工作用户
RUN useradd -m runner

WORKDIR /home/runner

USER runner

CMD ["python3"]
```

### 2.3 Java 镜像

```dockerfile
# oj-runner-java17/Dockerfile
FROM eclipse-temurin:17-jre-alpine

RUN addgroup -S runner && adduser -S runner -G runner

WORKDIR /home/runner

USER runner

CMD ["java"]
```

---

## 三、运行时隔离策略

### 3.1 cgroups 资源限制

```yaml
# docker-compose judge-worker 部分
judge-worker:
  image: oj-judge-worker:latest
  volumes:
    - /var/run/docker.sock:/var/run/docker.sock
  environment:
    - DOCKER_HOST=unix:///var/run/docker.sock
  deploy:
    resources:
      limits:
        cpus: '4'
        memory: 8G
```

### 3.2 容器运行参数

```go
func (r *DockerRunner) RunContainer(task *JudgeTask) (*RunResult, error) {
    // 计算资源限制
    timeLimit := time.Duration(task.Problem.TimeLimit) * time.Millisecond
    memLimit := int64(task.Problem.MemoryLimit) * 1024 * 1024 // MB -> bytes
    pidsLimit := int64(task.Language.PidsLimit)

    // 语言因子放大
    timeLimit = timeLimit * time.Duration(task.Language.TimeFactor*100)/100
    memLimit = memLimit * int64(task.Language.MemoryFactor*100)/100

    // 构建容器配置
    resp, err := r.client.ContainerCreate(r.ctx, &container.Config{
        Image:        task.Language.DockerImage,
        Tty:          false,
        AttachStdin:  false,
        AttachStdout: false,
        AttachStderr: false,
    }, &container.HostConfig{
        // 资源限制
        Resources: container.Resources{
            Memory:           memLimit,
            MemorySwap:       memLimit, // 禁用 swap
            CPUQuota:         int64(timeLimit.Milliseconds() * 1000 * 1000), // 100% CPU
            CPUPeriod:        100000, // 100ms
            PidsLimit:        &pidsLimit,
        },

        // 网络隔离 (禁止网络)
        NetworkMode: "none",

        // 只读挂载
        ReadonlyRootfs: true,

        // 临时写入 (tmpfs)
       Tmpfs: map[string]string{
            "/tmp": "size=256m,mode=1777",
            "/home/runner": "size=256m,mode=1777",
        },

        // 挂载测试数据 (只读)
        Binds: []string{
            fmt.Sprintf("%s:/data:ro", testDataPath),
            fmt.Sprintf("%s:/workspace:ro", workspacePath),
            fmt.Sprintf("%s/tmp:/tmp:rw", tmpPath),
        },

        // 安全选项
        SecurityOpt: []string{
            "no-new-privileges:true",
            "seccomp:default", // 使用默认 seccomp
        },

        // 自动清理
        AutoRemove: true,
    }, nil, "")

    return r.waitAndCollect(resp.ID, timeLimit)
}
```

### 3.3 网络隔离

```go
// 创建禁用网络的网络模式
func (r *DockerRunner) createIsolatedNetwork() error {
    _, err := r.client.NetworkCreate(r.ctx, "oj-isolated", &network.NetworkingConfig{
        Config: map[string]*network.EndpointSettings{
            "isolated": {
                NetworkMode: "none",
            },
        },
    })
    return err
}
```

### 3.4 只读文件系统

```dockerfile
# 容器内文件系统
/
├── bin/
├── usr/
├── etc/
├── lib/            (只读)
├── tmp/            (读写, tmpfs)
├── home/runner/    (读写, tmpfs)
└── data/           (只读挂载, 测试数据)
```

---

## 四、多语言支持实现

### 4.1 Language Registry 配置

```json
// languages 表中的配置
[
  {
    "name": "C++17",
    "slug": "cpp17",
    "source_filename": "main.cpp",
    "compile_cmd": ["g++", "-o", "main", "main.cpp", "-std=c++17", "-O2", "-Wall"],
    "compile_timeout": 10,
    "run_cmd": ["./main"],
    "docker_image": "oj-runner-cpp17",
    "time_factor": 1.0,
    "memory_factor": 1.0,
    "output_limit": 65536,
    "pids_limit": 64,
    "enabled": true
  },
  {
    "name": "Python 3.11",
    "slug": "py311",
    "source_filename": "main.py",
    "compile_cmd": null,
    "run_cmd": ["python3", "main.py"],
    "docker_image": "oj-runner-py311",
    "time_factor": 3.0,
    "memory_factor": 2.0,
    "output_limit": 65536,
    "pids_limit": 32,
    "enabled": true
  },
  {
    "name": "Java 17",
    "slug": "java17",
    "source_filename": "Main.java",
    "compile_cmd": ["javac", "Main.java"],
    "compile_timeout": 30,
    "run_cmd": ["java", "-cp", ".", "-Xmx256m", "Main"],
    "docker_image": "oj-runner-java17",
    "time_factor": 2.0,
    "memory_factor": 2.0,
    "output_limit": 65536,
    "pids_limit": 64,
    "enabled": true
  }
]
```

### 4.2 编译/运行命令构建

```go
type LanguageConfig struct {
    CompileCmd     []string `json:"compile_cmd"`
    RunCmd         []string `json:"run_cmd"`
    SourceFilename string   `json:"source_filename"`
    TimeFactor     float64  `json:"time_factor"`
    MemoryFactor   float64  `json:"memory_factor"`
}

func (c *LanguageConfig) BuildCompileCmd() []string {
    if len(c.CompileCmd) == 0 {
        return nil // 解释型语言
    }
    // 替换占位符
    cmd := make([]string, len(c.CompileCmd))
    for i, part := range c.CompileCmd {
        if part == "{source}" {
            cmd[i] = c.SourceFilename
        } else {
            cmd[i] = part
        }
    }
    return cmd
}

func (c *LanguageConfig) BuildRunCmd() []string {
    cmd := make([]string, len(c.RunCmd))
    for i, part := range c.RunCmd {
        if part == "{binary}" {
            // C++: ./main, Java: Main.class
            ext := filepath.Ext(c.SourceFilename)
            name := strings.TrimSuffix(c.SourceFilename, ext)
            if ext == ".java" {
                cmd[i] = name
            } else {
                cmd[i] = "./" + name
            }
        } else {
            cmd[i] = part
        }
    }
    return cmd
}
```

---

## 五、SPJ 判题支持

### 5.1 SPJ 配置

```go
type ProblemConfig struct {
    IsSPJ     bool   `json:"is_spj"`
    SPJLang   string `json:"spj_lang"`  // cpp17
    SPJCode   string `json:"spj_code"`  // MinIO URL
}
```

### 5.2 SPJ 运行

```go
func (r *DockerRunner) runSPJ(spjConfig *SPJConfig, input, output, expected string) (bool, string, error) {
    // 编译 SPJ
    spjImage := fmt.Sprintf("oj-runner-%s", spjConfig.SPJLang)
    // ...

    // 运行 SPJ
    cmd := []string{
        "./spj",
        inputFile,
        outputFile,
        expectedFile,
    }

    resp, err := r.client.ContainerCreate(..., &container.Config{
        Image: spjImage,
        Cmd:   cmd,
    }, ...)

    // SPJ 返回码: 0=AC, 1=WA, 其他=错误
    if resp.ExitCode == 0 {
        return true, "", nil
    }
    return false, "Special Judge: Wrong Answer", nil
}
```

---

## 六、安全加固

### 6.1 禁止特权模式

```yaml
# docker-compose.yml
security_opt:
  - no-new-privileges:true
```

### 6.2 资源配额

```yaml
ulimits:
  nproc: 64
  nofile:
    soft: 1024
    hard: 1024
```

### 6.3 Seccomp 限制

```json
// seccomp-profile.json (禁止危险系统调用)
{
  "defaultAction": "SCMP_ACT_ERRNO",
  "syscalls": [
    { "names": ["read", "write", ...], "action": "SCMP_ACT_ALLOW" },
    { "names": ["unshare", "clone", ...], "action": "SCMP_ACT_ERRNO" }
  ]
}
```

---

## 七、镜像构建脚本

```bash
# build-images.sh
#!/bin/bash
set -e

REGISTRY="oj-runner"
TAG="latest"

for lang in cpp17 c11 go122 py311 java17; do
    echo "Building $lang..."
    docker build -t ${REGISTRY}-${lang}:${TAG} ./images/${lang}
done

# 推送
# docker push ${REGISTRY}-${lang}:${TAG}
```
