import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:latlong2/latlong.dart';
import 'package:geolocator/geolocator.dart';
import 'dart:async';

import '../models/ping.dart';
import '../api/ping_client.dart';
import '../config.dart';
import '../services/identity_service.dart';
import '../theme/ye_ke.dart';

class MapScreen extends StatefulWidget {
  const MapScreen({super.key});

  @override
  State<MapScreen> createState() => _MapScreenState();
}

class _MapScreenState extends State<MapScreen> {
  final _mapController = MapController();
  final _client = PingClient(baseUrl: apiBaseUrl);

  var _busy = false;
  List<Ping> _pings = [];
  Timer? _debounce;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _centerOnMe());
  }

  Future<void> _centerOnMe() async {
    try {
      final pos = await Geolocator.getCurrentPosition();
      _mapController.move(LatLng(pos.latitude, pos.longitude), 15);
    } catch (_) {}
  }

  void _onPositionChanged(MapCamera camera, bool hasGesture) {
    _debounce?.cancel();
    _debounce = Timer(const Duration(milliseconds: 400), _refresh);
  }

  Future<void> _refresh() async {
    try {
      final token = await IdentityService().token();
      final pings = await _client.inBounds(
        _mapController.camera.visibleBounds,
        token: token,
      );

      if (!mounted) return;
      setState(() => _pings = pings);
    } catch (e) {
      debugPrint('refresh failed: $e');
    }
  }

  Future<void> _ping() async {
    if (_busy) return;
    setState(() => _busy = true);
    HapticFeedback.mediumImpact();

    try {
      final token = await IdentityService().token();
      if (token == null) return; //Save to local DB here

      final pos = await Geolocator.getCurrentPosition();
      final p = await _client.create(
        kind: 'roadblock',
        lat: pos.latitude,
        lng: pos.longitude,
        token: token,
      );
      if (p == null || !mounted) return;

      _mapController.move(LatLng(p.lat, p.lng), 15);
      await _refresh();
    } on PingCooldownException catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(e.message)));
      }
    } catch (_) {
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  void dispose() {
    _debounce?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: YeKe.paper,
      body: FlutterMap(
        mapController: _mapController,
        options: MapOptions(
          onMapReady: _refresh,
          initialCenter: LatLng(3.1390, 101.6869),
          initialZoom: 12,
          onPositionChanged: _onPositionChanged,
        ),
        children: [
          TileLayer(
            urlTemplate: 'https://tile.openstreetmap.org/{z}/{x}/{y}.png',
            userAgentPackageName: 'com.byakugan.byakugan',
          ),
        ],
      ),
    );
  }
}
