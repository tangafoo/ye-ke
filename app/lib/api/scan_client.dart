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

  /// Street photos around the tap, for the loading strip. Served from our DB,
  /// or fetched from Mapillary once per block, so it returns fast.
  Future<Map<String, dynamic>?> preview(LatLng p) async {
    final uri = Uri.parse('$baseUrl/scan/preview').replace(
      queryParameters: {'lat': '${p.latitude}', 'lng': '${p.longitude}'},
    );

    final res = await _http.get(uri).timeout(const Duration(seconds: 25));
    if (res.statusCode != 200) return null;
    return jsonDecode(res.body) as Map<String, dynamic>;
  }
}
