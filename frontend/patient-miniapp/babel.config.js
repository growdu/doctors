// babel.config.js
//
// Babel 配置 —— 给 babel-jest 提供 ES2020+ → ES2018 转译
//（jest 默认走 CommonJS；本项目源码用 ES Modules，必须 babel 转译）。
//
// 设计要点（plan Task 9）：
//   - 仅启用 @babel/preset-env：覆盖 import/export + 箭头函数 + 解构等 ES2020 语法
//   - targets: { node: 'current' }：跑在 Node 18 LTS（uni-app 最低要求）
//   - 不启 @babel/preset-react / typescript：当前源码为 JS，TS 单测走 ts-jest（后续 plan）
//
// 安装依赖（任务外步骤，CI 阶段会跑）：
//   npm i -D @babel/core @babel/preset-env babel-jest
module.exports = {
  presets: [
    [
      '@babel/preset-env',
      {
        targets: { node: 'current' },
        // jest 跑在 Node 而非浏览器；modules 走 'commonjs' 让 require() 兼容
        modules: 'commonjs',
        // 单元测试更看重栈可读性，关掉 minify
        comments: true,
      },
    ],
  ],
};