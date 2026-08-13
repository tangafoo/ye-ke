import 'dart:convert';

import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:http/http.dart' as http;

import '../config.dart';

/// Anonymous-first identity. First launch registers a user with the backend —
/// no signup, no screen — and keeps the id + bearer token in secure storage.
/// OAuth linking later attaches credentials to this same user id.
///
/// Every failure here is silent by design: the app must be fully usable
/// unidentified, so a failed registration just retries on the next launch.
class IdentityService {
  static const _kUserId = 'yk_user_id';
  static const _kToken = 'yk_token';

  static const _storage = FlutterSecureStorage();

  Future<void> ensureRegistered() async {
    try {
      if (await _storage.read(key: _kToken) != null) return;

      final res = await http
          .post(Uri.parse('$apiBaseUrl/users/anonymous'))
          .timeout(const Duration(seconds: 10));
      if (res.statusCode != 200) return;

      final body = jsonDecode(res.body) as Map<String, dynamic>;
      final userId = body['user_id'] as String?;
      final token = body['token'] as String?;
      if (userId == null || token == null) return;

      await _storage.write(key: _kUserId, value: userId);
      await _storage.write(key: _kToken, value: token);
    } catch (_) {
      // Offline or server down on first launch — never block the app.
    }
  }

  /// The bearer token for API calls, or null while unregistered.
  Future<String?> token() => _storage.read(key: _kToken);
}
