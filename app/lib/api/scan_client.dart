import 'dart:convert';

import 'package:http/http.dart' as http;
import 'package:latlong2/latlong.dart';

class ScanClient {
  final String baseUrl;
  final http.Client _http;

  ScanClient({required this.baseUrl, http.Client? client})
    : _http = client ?? http.Client();

  Future<Map<String, dynamic>?> scan(LatLng p, {double radiusM = 500}) async {
    final uri = Uri.parse('$baseUrl/scan').replace(
      queryParameters: {
        'lat': '${p.latitude}',
        'lng': '${p.longitude}',
        'radius': '$radiusM',
      },
    );

    // a cold scan sweeps OSM + news live (~30 s); warm scans read the DB instantly
    final res = await _http.get(uri).timeout(const Duration(seconds: 60));
    if (res.statusCode != 200) return null;
    return jsonDecode(res.body) as Map<String, dynamic>;
  }
}
