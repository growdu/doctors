// jest.config.js
//
// Jest 配置 —— 覆盖本目录全部 `*.test.js` 与 `__tests__/**/*.test.js`。
//
// 设计要点（plan Task 9）：
//   - testEnvironment: 'jsdom'：uni 业务对象（uni.getStorageSync 等）依赖 DOM 全局
//   - moduleNameMapper: '@/' → 'src/'：与 Vite / uni-app 约定一致
//   - transform: babel-jest 处理 ES2020 模块语法（import/export）
//   - testPathIgnorePatterns: 忽略 node_modules 与 uni-app 输出 dist/unpackage
//
// 注：当前 .vue 单测改走 e2e（H5 Playwright），故 jest 不挂 vue-jest。
//     vue-jest 需要时由后续 plan（Task 8/9）按需引入。

module.exports = {
  testEnvironment: 'jsdom',
  roots: ['<rootDir>/src', '<rootDir>/__tests__'],
  moduleFileExtensions: ['js', 'json'],
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/src/$1',
  },
  transform: {
    '^.+\\.js$': 'babel-jest',
  },
  testPathIgnorePatterns: [
    '/node_modules/',
    '/dist/',
    '/unpackage/',
    '/coverage/',
  ],
  transformIgnorePatterns: ['/node_modules/'],
  collectCoverageFrom: [
    'src/**/*.{js,vue}',
    '!src/**/*.test.js',
  ],
  testMatch: [
    '**/__tests__/**/*.test.[jt]s?(x)',
    '**/?(*.)+(spec|test).[jt]s?(x)',
  ],
};