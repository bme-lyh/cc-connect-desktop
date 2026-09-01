# CC-Connect Desktop

一个更容易上手的 Windows 桌面版 cc-connect。让本机上的 Codex、Pi 等编程代理，通过飞书、Telegram、微信等聊天工具随时为你工作。

A simpler Windows desktop edition of cc-connect. It lets you use local coding agents such as Codex and Pi from Feishu/Lark, Telegram, WeChat, and other chat apps.

> 本项目派生自 [chenhg5/cc-connect](https://github.com/chenhg5/cc-connect)，保留原有 `cc-connect` 命令、Go module 路径和高级配置能力。
>
> This project is derived from [chenhg5/cc-connect](https://github.com/chenhg5/cc-connect) and keeps the original `cc-connect` command, Go module path, and advanced configuration support.

## 中文说明

简单来说：程序运行在你的 Windows 电脑上，聊天机器人负责把消息转给本地编程代理，再把回答发回聊天窗口。代码、项目目录和代理进程都留在本机。

这个版本主要简化了第一次使用：

- 不必先手写 `config.toml`，可以零配置启动。
- 首次向导会依次选择代理、机器人名称、项目文件夹和聊天平台。
- 支持 Windows 文件夹选择器和本机模型发现。
- 可以在网页里启动、停止、编辑机器人，也可以手动退出程序。
- 可以选择是否在回答底部显示运行信息。
- Windows 下的机器人密钥保存在 Credential Manager，不以明文写入配置文件。
- 保留原项目的高级配置、平台适配和命令行用法。

## 操作截图 / Screenshots

在聊天窗口发送需求，本机代理处理后会在同一会话中回复。

Send a request in your chat app and receive the local agent's answer in the same conversation.

<table>
  <tr>
    <th>飞书 / Feishu</th>
    <th>Telegram</th>
    <th>微信 / WeChat</th>
  </tr>
  <tr>
    <td><img src="docs/images/screenshot/cc-connect-lark.JPG" alt="CC-Connect in Feishu" /></td>
    <td><img src="docs/images/screenshot/cc-connect-telegram.JPG" alt="CC-Connect in Telegram" /></td>
    <td><img src="docs/images/screenshot/cc-connect-wechat.JPG" alt="CC-Connect in WeChat" /></td>
  </tr>
</table>

## Windows 快速开始

### 使用便携包（推荐）

1. 从 GitHub Releases 下载 Windows 便携包并解压。
2. 运行解压目录根部的 `install-windows.ps1`（不是源码仓库里的同名脚本）。
3. 打开桌面或开始菜单中的 **CC-Connect**。
4. 浏览器会打开本地管理页，按照五步向导创建第一个机器人。

### 从源码安装

源码仓库不会提交 `cc-connect.exe`。已安装 Node.js 与 Go 时，可在仓库根目录运行：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\install-windows.ps1
```

脚本会先构建网页和 `build\cc-connect.exe`，再完成安装。如果 Go 不在 `PATH`，可额外传入 `-GoPath "C:\path\to\go.exe"`。

详细中文用法见 [docs/usage.zh-CN.md](docs/usage.zh-CN.md)。

## English

In plain English: CC-Connect runs on your Windows PC. A chat bot forwards your messages to a local coding agent and sends its answers back. Your code, workspace, and agent process stay on your computer.

This edition focuses on an easier first run:

- Start without writing `config.toml` first.
- Use a guided setup for the agent, bot name, workspace, and chat platform.
- Pick a Windows folder and discover models available on the machine.
- Start, stop, edit, or quit local bots from the browser page.
- Choose whether runtime details appear below answers.
- Store bot secrets in Windows Credential Manager instead of plain-text configuration.
- Keep the upstream project's advanced configuration, platforms, and CLI commands.

### Windows quick start

#### Portable package (recommended)

1. Download and extract the Windows portable package from GitHub Releases.
2. Run `install-windows.ps1` from the root of the extracted package, not the similarly named script in a source checkout.
3. Open **CC-Connect** from the Desktop or Start menu.
4. Follow the five-step wizard in the local management page.

#### Install from source

The source repository does not include `cc-connect.exe`. With Node.js and Go installed, run this from the repository root:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\install-windows.ps1
```

The script builds the web UI and `build\cc-connect.exe` before installing. If Go is not on `PATH`, also pass `-GoPath "C:\path\to\go.exe"`.

See [docs/usage.md](docs/usage.md) for detailed usage.

## Security / 安全说明

- 不要提交 `config.toml`、Token、日志、会话目录或构建产物。
- 本地运行配置通常位于用户目录，不属于仓库内容。
- Never commit `config.toml`, tokens, logs, session data, or build artifacts.

## Upstream and license

- Upstream: [chenhg5/cc-connect](https://github.com/chenhg5/cc-connect)
- License: [MIT](LICENSE)
