import 'dart:io' show Platform;

import 'package:flutter/foundation.dart' show kIsWeb;

const _apiOverride = String.fromEnvironment('API_BASE_URL');

String get apiBaseUrl {
  if (_apiOverride.isNotEmpty) return _apiOverride;
  if (kIsWeb) return 'http://localhost:8080';
  if (Platform.isAndroid) return 'http://10.0.2.2:8080';

  return 'http://localhost:8080';
}
