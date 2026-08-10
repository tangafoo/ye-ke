import 'dart:io' show Platform;

import 'package:flutter/foundation.dart' show kIsWeb;

String get apiBaseUrl {
  if (kIsWeb) return 'http://localhost:8080';
  if (Platform.isAndroid) return 'http://10.0.2.2:8080';
  return 'http://localhost:8080';
}
