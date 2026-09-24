// lib/core/theme.dart
//
// Material 3 主题 —— 主色 #1989FA（spec 全局配色规范）。
import 'package:flutter/material.dart';

/// 主色 —— spec 全局规范（lib/api 与小程序端共用同一蓝色品牌色）。
const Color kBrandPrimary = Color(0xFF1989FA);

/// 完整 ThemeData（Material 3）。
final ThemeData appTheme = ThemeData(
  useMaterial3: true,
  colorScheme: ColorScheme.fromSeed(
    seedColor: kBrandPrimary,
    brightness: Brightness.light,
  ),
  appBarTheme: const AppBarTheme(
    centerTitle: true,
    elevation: 0,
  ),
);