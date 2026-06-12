# mimi-router 前端界面

本项目是 mimi-router 的前端界面（w-run 二次开发版）。

## 使用的开源项目

基于以下开源项目开发：

- [Berry Free React Admin Template](https://github.com/codedthemes/berry-free-react-admin-template)
- 原版 One API 前端：[songquanpeng/one-api](https://github.com/songquanpeng/one-api)

## 开发说明

当添加新的渠道时，需要修改以下文件：

1. `web/default/src/constants/ChannelConstants.js`

在该文件中的 `CHANNEL_OPTIONS` 添加新的渠道：

```js
export const CHANNEL_OPTIONS = {
  // key 为渠道ID
  1: {
    key: 1, // 渠道ID
    text: "OpenAI", // 渠道名称
    value: 1, // 渠道ID
    color: "primary", // 渠道列表显示的颜色
  },
};
```

2. `web/default/src/views/Channel/type/Config.js`

在该文件中的 `typeConfig` 添加新的渠道配置，如无需配置可不添加：

```js
const typeConfig = {
  3: {
    inputLabel: {
      base_url: "AZURE_OPENAI_ENDPOINT",
      other: "默认 API 版本",
    },
    prompt: {
      base_url: "请填写AZURE_OPENAI_ENDPOINT",
      other: "请输入默认API版本，例如：2024-03-01-preview",
    },
    modelGroup: "openai",
  },
};
```

## 构建

```bash
npm install
npm run build
```

构建产物位于 `../build/default/`（由 package.json 的 build 脚本自动移动）。

## 许可证

本项目遵循 MIT 协议。
