import 'dart:convert';

import 'package:flutter_map/flutter_map.dart';
import 'package:http/http.dart' as http;

import '../models/ping.dart';

class PingCooldownException implements Exception {
  final String message;
  PingCooldownException(this.message);
}

class PingClient {
  final String baseUrl;
  final http.Client _http;

  PingClient({required this.baseUrl, http.Client? client})
    : _http = client ?? http.Client();

  Future<Ping?> create({
    required String kind,
    required double lat,
    required double lng,
    required String token,
  }) async {
    final res = await _http
        .post(
          Uri.parse('$baseUrl/pings'),
          headers: {
            'Content-Type': 'application/json',
            'Authorization': 'Bearer $token',
          },
          body: jsonEncode({'kind': kind, 'lat': lat, 'lng': lng}),
        )
        .timeout(const Duration(seconds: 10));

    if (res.statusCode == 429) throw PingCooldownException(res.body.trim());
    if (res.statusCode != 201) return null;

    return Ping.fromJson(jsonDecode(res.body) as Map<String, dynamic>);
  }

  Future<List<Ping>> inBounds(LatLngBounds b, {String? token}) async {
    final uri = Uri.parse('$baseUrl/pings').replace(
      queryParameters: {'bbox': '${b.south},${b.west},${b.north},${b.east}'},
    );

    final res = await _http
        .get(
          uri,
          headers: {if (token != null) 'Authorization': 'Bearer $token'},
        )
        .timeout(const Duration(seconds: 10));

    if (res.statusCode != 200) return const [];
    final body = jsonDecode(res.body) as Map<String, dynamic>;
    final list = body['pings'] as List<dynamic>? ?? const [];

    return [for (final j in list) Ping.fromJson(j as Map<String, dynamic>)];
  }
}
