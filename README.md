# DaoPowerplan

一站式 Windows 电源计划管理工具，支持预设模式一键切换、自定义电源计划创建与详细参数调节。

## 功能特性

- **预设电源模式**：内置游戏模式（Intel/AMD）、办公模式（Intel/AMD），一键创建并应用
- **系统计划管理**：查看、切换、删除系统电源计划（平衡、高性能、节能、卓越性能等）
- **详细参数编辑**：类似 PowerSettingsExplorer 的设置编辑面板，支持所有电源设置项的查看与修改
- **预设选项下拉**：有预设值的设置项自动显示下拉选择，数值型设置项支持直接输入
- **防重复创建**：创建预设模式前自动检测，避免重复创建同名计划
- **中文界面**：全中文操作界面，操作直观

## 预设模式说明

| 模式 | 适用平台 | 说明 |
|---|---|---|
| 游戏模式 · Intel | Intel CPU | 性能拉满，大核优先调度，处理器 100% 运行，PCIe/USB 节能全部关闭 |
| 游戏模式 · AMD | AMD CPU | 全核满载，极致性能，处理器 100% 运行，PCIe/USB 节能全部关闭 |
| 办公模式 · Intel | Intel CPU | 平衡省电，智能调度，处理器最大 70%，硬盘/显示器自动休眠 |
| 办公模式 · AMD | AMD CPU | 均衡能效，安静办公，处理器最大 50%，硬盘/显示器自动休眠 |

### 游戏模式核心优化项

- 处理器最小/最大状态：100%
- 性能提升模式：激进
- 性能升降策略：Rocket（火箭）
- 系统散热方式：主动
- PCIe 链接状态电源管理：关闭
- USB 选择性暂停：禁用
- 核心停车：禁用

## 系统要求

- Windows 10 / Windows 11（64 位）
- 管理员权限（修改电源计划需要）

## 编译方式

### 环境要求

- Go 1.22+
- Wails CLI v2.8+

### 编译命令

```bash
# 安装依赖
go mod tidy

# 编译为 Windows 单文件 EXE
go build -tags "production" -ldflags "-H windowsgui" -o DaoPowerplan.exe .
```

## 项目结构

```
DaoPowerplan/
├── main.go              # 程序入口，Wails 初始化，GBK 编码转换
├── app.go               # 核心逻辑：电源计划管理、设置查询修改、预设模式
├── go.mod               # Go 依赖配置
├── go.sum               # 依赖校验
├── wails.json           # Wails 配置
├── rsrc.syso            # 图标资源
└── frontend/
    └── dist/
        └── index.html   # 前端界面
```

## 技术栈

- **前端**：原生 HTML / CSS / JavaScript
- **后端**：Go + Wails v2
- **电源管理**：Windows powercfg 命令行接口
- **编码处理**：golang.org/x/text（GBK → UTF-8 转换）

## 作者

Daomak

- 作者主页：www.daomak.com
