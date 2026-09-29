// lib/services/token_storage.dart
//
// Token 持久化 —— flutter_secure_storage 封装（spec §1.2：JWT 用密钥存储）。
// v1：真实实现走 `_SecureTokenStorage`；测试用 `TokenStorage.forTest()` 返回 in-memory fake。
//
// 设计：抽象类 + factory 模式，让上层（auth_provider / api_client）只依赖抽象，
//      测试可注入 fake；与 `kStorageKeyToken` 常量对齐 lib/core/constants.dart。
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import '../core/constants.dart';

/// Token 持久化接口。
abstract class TokenStorage {
  /// 写入 access token（覆盖）。
  Future<void> write(String token);

  /// 读取 access token；无则返回 null。
  Future<String?> read();

  /// 删除 access token（用于 logout / 401 cleanup）。
  Future<void> delete();

  /// 注入真实 flutter_secure_storage（生产 / 集成测试）。
  factory TokenStorage(FlutterSecureStorage storage) = _SecureTokenStorage;

  /// 注入 in-memory fake（单测用，不依赖 platform channel）。
  factory TokenStorage.forTest() = _FakeTokenStorage;
}

/// 真实实现：使用 `flutter_secure_storage`（Keychain / EncryptedSharedPreferences）。
class _SecureTokenStorage implements TokenStorage {
  final FlutterSecureStorage _s;

  _SecureTokenStorage(this._s);

  @override
  Future<void> write(String token) =>
      _s.write(key: kStorageKeyToken, value: token);

  @override
  Future<String?> read() => _s.read(key: kStorageKeyToken);

  @override
  Future<void> delete() => _s.delete(key: kStorageKeyToken);
}

/// 内存实现（测试用）—— 单线程无并发问题，state 在 isolate 内自包含。
class _FakeTokenStorage implements TokenStorage {
  String? _t;

  @override
  Future<void> write(String token) async {
    _t = token;
  }

  @override
  Future<String?> read() async => _t;

  @override
  Future<void> delete() async {
    _t = null;
  }
}