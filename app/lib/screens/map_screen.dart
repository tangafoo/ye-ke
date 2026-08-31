import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:latlong2/latlong.dart';
import 'package:geolocator/geolocator.dart';
import 'dart:async';

import '../models/ping.dart';
import '../widgets/radar_view.dart';
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

  var _radarOpen = true;
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
      _mapController.move(LatLng(pos.latitude, pos.longitude), 30);
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

      _mapController.move(LatLng(p.lat, p.lng), 30);
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

  var _rangeControlsOpened = false;

  @override
  Widget build(BuildContext context) {
    return ColoredBox(
      color: YeKe.paint,
      child: SafeArea(
        bottom: false,
        child: Column(
          children: [
            AnimatedContainer(
              duration: const Duration(milliseconds: 250),
              curve: Curves.easeInOut,
              height: _radarOpen ? MediaQuery.sizeOf(context).height / 3 : 0,
              width: double.infinity,
              child: ClipRect(
                child: Row(
                  children: [
                    Expanded(
                      flex: 1,
                      child: Column(
                        children: [
                          if (_rangeControlsOpened)
                            Expanded(
                              flex: 3,
                              child: Text('Range', textAlign: TextAlign.center),
                            )
                          else ...[
                            Expanded(
                              child: Container(
                                height: 20,
                                width: double.infinity,
                                decoration: BoxDecoration(
                                  color: YeKe.bg1,
                                  border: BoxBorder.fromLTRB(
                                    bottom: BorderSide(color: YeKe.night),
                                  ),
                                ),
                                child: const Icon(
                                  Icons.newspaper,
                                  color: YeKe.paper,
                                  size: 42,
                                ),
                              ),
                            ),

                            Expanded(
                              child: Container(
                                height: 20,
                                width: double.infinity,
                                decoration: BoxDecoration(
                                  color: YeKe.scaley,
                                  border: BoxBorder.fromLTRB(
                                    bottom: BorderSide(color: YeKe.night),
                                  ),
                                ),
                                child: const Icon(
                                  Icons.remove_red_eye,
                                  color: YeKe.paper,
                                  size: 42,
                                ),
                              ),
                            ),
                            Expanded(
                              child: Container(
                                height: 20,
                                width: double.infinity,
                                decoration: BoxDecoration(
                                  color: YeKe.scaley,
                                  border: BoxBorder.fromLTRB(
                                    bottom: BorderSide(color: YeKe.night),
                                  ),
                                ),
                                child: const Icon(
                                  Icons.houseboat,
                                  color: YeKe.paper,
                                  size: 42,
                                ),
                              ),
                            ),
                          ],

                          Expanded(
                            child: GestureDetector(
                              onTap: () => setState(
                                () => _rangeControlsOpened =
                                    !_rangeControlsOpened,
                              ),
                              child: Container(
                                height: 20,
                                width: double.infinity,
                                decoration: BoxDecoration(
                                  color: YeKe.scaley,
                                  border: BoxBorder.fromLTRB(
                                    bottom: BorderSide(color: YeKe.night),
                                  ),
                                ),
                                child: const Icon(
                                  Icons.engineering,
                                  color: YeKe.paper,
                                  size: 42,
                                ),
                              ),
                            ),
                          ),
                        ],
                      ),
                    ),
                    Expanded(
                      flex: 5,
                      child: Column(
                        children: [
                          Container(
                            width: double.infinity,
                            color: YeKe.cement,
                            alignment: Alignment.center,
                            child: Text(
                              'Scanner 2.0 ACAB ANTIFA',
                              style: const TextStyle(
                                fontFamily: 'Faculty',
                                color: YeKe.bg1,
                              ),
                            ),
                          ),
                          Expanded(
                            child: Container(
                              decoration: const BoxDecoration(
                                color: YeKe.cement,
                              ),
                              padding: const EdgeInsets.fromLTRB(16, 0, 14, 16),
                              child: Container(
                                decoration: BoxDecoration(
                                  color: YeKe.night,
                                  borderRadius: BorderRadius.circular(16),
                                  boxShadow: [
                                    BoxShadow(
                                      color: YeKe.scaley,
                                      offset: const Offset(6, 0),
                                    ),
                                  ],
                                ),
                                child: RadarView(),
                              ),
                            ),
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
            ),
            GestureDetector(
              onTap: () => setState(() => _radarOpen = !_radarOpen),
              child: Container(
                width: double.infinity,
                color: YeKe.dusky,
                padding: const EdgeInsets.symmetric(vertical: 2),
                child: Icon(
                  _radarOpen
                      ? Icons.keyboard_arrow_up
                      : Icons.keyboard_arrow_down,
                  size: 16,
                  color: YeKe.sub,
                ),
              ),
            ),
            Expanded(
              child: FlutterMap(
                mapController: _mapController,
                options: MapOptions(
                  onMapReady: _refresh,
                  initialCenter: LatLng(3.1390, 101.6869),
                  initialZoom: 12,
                  onPositionChanged: _onPositionChanged,
                ),
                children: [
                  TileLayer(
                    urlTemplate:
                        'https://tile.openstreetmap.org/{z}/{x}/{y}.png',
                    userAgentPackageName: 'com.byakugan.byakugan',
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
