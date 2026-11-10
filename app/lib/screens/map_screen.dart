import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:latlong2/latlong.dart';
import 'package:geolocator/geolocator.dart';
import 'package:country_flags/country_flags.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'dart:async';
import 'dart:ui' show ImageFilter;

import '../models/ping.dart';
import '../widgets/radar_view.dart';
import '../api/ping_client.dart';
import '../config.dart';
import '../services/identity_service.dart';
import '../theme/ye_ke.dart';

enum _City {
  kl('KL', 'MY', LatLng(3.1390, 101.6869)),
  leipzig('Leipzig', 'DE', LatLng(51.3397, 12.3731)),
  brisbane('Brisbane', 'AU', LatLng(-27.4698, 153.0251)),
  bordeaux('Bordeaux', 'FR', LatLng(44.8378, -0.5792));

  const _City(this.label, this.countryCode, this.center);
  final String label;
  final String countryCode;
  final LatLng center;
}

enum _PanelTab {
  copStories(asset: 'assets/images/mantis-cop.webp'),
  emitters(asset: 'assets/images/climate-corpo.webp'),
  houseboat(icon: Icons.houseboat);

  const _PanelTab({this.icon, this.asset});
  final IconData? icon;
  final String? asset;
}

class MapScreen extends StatefulWidget {
  const MapScreen({super.key});

  @override
  State<MapScreen> createState() => _MapScreenState();
}

class _MapScreenState extends State<MapScreen>
    with SingleTickerProviderStateMixin {
  final _mapController = MapController();
  final _client = PingClient(baseUrl: apiBaseUrl);

  var _covered = false;

  late final _moveAnim = AnimationController(
    vsync: this,
    duration: const Duration(milliseconds: 888),
  );
  VoidCallback? _moveTick;

  void _animatedMove(LatLng dest, double destZoom) {
    final cam = _mapController.camera;
    final lat = Tween<double>(begin: cam.center.latitude, end: dest.latitude);
    final lng = Tween<double>(begin: cam.center.longitude, end: dest.longitude);
    final zoom = Tween<double>(begin: cam.zoom, end: destZoom);

    if (_moveTick != null) _moveAnim.removeListener(_moveTick!);
    _moveTick = () {
      final t = Curves.easeInCubic.transform(_moveAnim.value);
      _mapController.move(
        LatLng(lat.transform(t), lng.transform(t)),
        zoom.transform(t),
      );
    };

    _moveAnim
      ..addListener(_moveTick!)
      ..forward(from: 0);
  }

  var _radarOpen = true;
  var _busy = false;
  List<Ping> _pings = [];
  Timer? _debounce;

  static const _kLat = 'map_lat';
  static const _kLng = 'map_lng';
  static const _kCity = 'map_city';

  var _city = _City.kl;

  LatLng? _start;
  LatLng? _currentLoc;
  StreamSubscription<Position>? _currentLocSub;

  final double _startZoom = 12;
  var _dragRegistered = false;

  Timer? _citySelectorTimer;

  @override
  void initState() {
    super.initState();
    _resolveStart();
  }

  Future<Position?> _lastKnown() async {
    try {
      // I guess this is a separate thing from the 4 LatLng selector. preserve this.
      return await Geolocator.getLastKnownPosition();
    } catch (_) {
      return null;
    }
  }

  Future<void> _resolveStart() async {
    LatLng? center;

    try {
      final p = await Geolocator.checkPermission();
      if (p == LocationPermission.whileInUse ||
          p == LocationPermission.always) {
        final pos =
            await _lastKnown() ??
            await Geolocator.getCurrentPosition(
              locationSettings: const LocationSettings(
                timeLimit: Duration(seconds: 5),
              ),
            );

        center = LatLng(pos.latitude, pos.longitude);
        _currentLoc = center;
        _currentLocSub =
            Geolocator.getPositionStream(
              locationSettings: const LocationSettings(distanceFilter: 5),
            ).listen((pos) {
              if (mounted) {
                setState(
                  () => _currentLoc = LatLng(pos.latitude, pos.longitude),
                );
              }
            }, onError: (_) {});
      }
    } catch (e) {
      debugPrint('location permission not granted: $e');
    }

    final prefs = await SharedPreferences.getInstance();
    if (center == null) {
      final lat = prefs.getDouble(_kLat);
      final lng = prefs.getDouble(_kLng);
      if (lat != null && lng != null) {
        center = LatLng(lat, lng);
      }
    }

    if (!mounted) return;
    final savedCity = _City.values.asNameMap()[prefs.getString(_kCity)];
    setState(() {
      _start = center ?? savedCity?.center ?? _City.kl.center;
      if (savedCity != null) _city = savedCity;
    });
  }

  void _centerOnMe() {
    if (_currentLoc == null) return;
    HapticFeedback.selectionClick();
    _animatedMove(_currentLoc!, 18);
  }

  void _onPositionChanged(MapCamera camera, bool hasGesture) {
    _debounce?.cancel();
    _debounce = Timer(const Duration(milliseconds: 400), _refresh);
  }

  void _onMapEvent(MapEvent event) {
    if (event.source != MapEventSource.onDrag) return;

    if (!_dragRegistered) setState(() => _dragRegistered = true);
    _citySelectorTimer?.cancel();
    _citySelectorTimer = Timer(const Duration(milliseconds: 900), () {
      if (mounted) setState(() => _dragRegistered = false);
    });
  }

  void _jumpTo(_City city) {
    HapticFeedback.selectionClick();
    setState(() {
      _city = city;
      _covered = true;
    });
    SharedPreferences.getInstance().then((p) => p.setString(_kCity, city.name));
  }

  void _onCoverFaded() {
    if (!_covered) return;
    _mapController.move(_city.center, _startZoom);
    setState(() => _covered = false);
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

      _mapController.move(LatLng(p.lat, p.lng), 18);
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
    _citySelectorTimer?.cancel();
    _currentLocSub?.cancel();
    _moveAnim.dispose();

    super.dispose();
  }

  var _rangeControlsOpened = false;
  var _tab = _PanelTab.copStories;

  Widget _tabButton(_PanelTab tab) {
    return Expanded(
      child: GestureDetector(
        onTap: () => setState(() => _tab = tab),
        child: Container(
          height: 20,
          width: double.infinity,
          decoration: BoxDecoration(
            color: _tab == tab ? YeKe.bg1 : YeKe.scaley,
            border: BoxBorder.fromLTRB(bottom: BorderSide(color: YeKe.night)),
          ),
          child: tab.asset != null
              ? AnimatedPadding(
                  curve: Curves.easeOutExpo,
                  duration: const Duration(milliseconds: 200),
                  padding: EdgeInsets.all(_tab == tab ? 8 : 12),
                  child: Image.asset(tab.asset!, width: 42, height: 42),
                )
              : Icon(tab.icon, color: YeKe.paper, size: 42),
        ),
      ),
    );
  }

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
                          else
                            for (final tab in _PanelTab.values) _tabButton(tab),

                          Expanded(
                            key: const ValueKey('radar-settings'),
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
                                child: AnimatedPadding(
                                  curve: Curves.easeOutExpo,
                                  duration: const Duration(milliseconds: 200),
                                  padding: EdgeInsets.all(
                                    _rangeControlsOpened ? 8 : 12,
                                  ),
                                  child: Image.asset(
                                    'assets/images/radar-settings.webp',
                                    width: 42,
                                    height: 42,
                                  ),
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
                              'Scanner 1.0 ACAB',
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
                                child: AnimatedSwitcher(
                                  duration: const Duration(milliseconds: 250),
                                  child: switch (_tab) {
                                    _PanelTab.copStories => const RadarView(
                                      key: ValueKey(_PanelTab.copStories),
                                    ),
                                    _PanelTab.emitters => DecoratedBox(
                                      key: const ValueKey(_PanelTab.emitters),
                                      decoration: BoxDecoration(
                                        color: const Color(0xFF071C21),
                                        borderRadius: BorderRadius.circular(16),
                                      ),
                                      child: const RadarView(
                                        gridColor: YeKe.oddish,
                                        sweepColor: YeKe.stand,
                                      ),
                                    ),
                                    _PanelTab.houseboat => const Center(
                                      key: ValueKey(_PanelTab.houseboat),
                                      child: Text(
                                        'Houseboat',
                                        style: TextStyle(color: YeKe.paper),
                                      ),
                                    ),
                                  },
                                ),
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
              child: _start == null
                  ? const Center(
                      child: CircularProgressIndicator(color: YeKe.rotom),
                    )
                  : Stack(
                      children: [
                        FlutterMap(
                          mapController: _mapController,
                          options: MapOptions(
                            onMapReady: _refresh,
                            initialCenter: _start!,
                            initialZoom: _startZoom,
                            onMapEvent: _onMapEvent,
                            maxZoom: 22,
                            onPositionChanged: _onPositionChanged,
                          ),
                          children: [
                            TileLayer(
                              urlTemplate:
                                  'https://tile.openstreetmap.org/{z}/{x}/{y}.png',
                              userAgentPackageName: 'com.byakugan.byakugan',
                            ),
                            if (_currentLoc != null)
                              MarkerLayer(
                                markers: [
                                  Marker(
                                    point: _currentLoc!,
                                    width: 70,
                                    height: 70,
                                    child: const _PulseDot(),
                                  ),
                                ],
                              ),
                          ],
                        ),
                        Positioned.fill(
                          child: IgnorePointer(
                            child: AnimatedOpacity(
                              opacity: _covered ? 1 : 0,
                              curve: Curves.easeOut,
                              duration: Duration(milliseconds: 420),
                              onEnd: _onCoverFaded,
                              child: const ColoredBox(color: YeKe.night),
                            ),
                          ),
                        ),
                        Positioned(
                          right: 12,
                          bottom: 72,
                          child: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              IgnorePointer(
                                ignoring: _dragRegistered,
                                child: AnimatedOpacity(
                                  opacity: _dragRegistered ? 0 : 1,
                                  duration: const Duration(seconds: 1),
                                  child: Directionality(
                                    textDirection: TextDirection.rtl,
                                    child: MenuAnchor(
                                      style: const MenuStyle(
                                        backgroundColor: WidgetStatePropertyAll(
                                          Colors.transparent,
                                        ),
                                        elevation: WidgetStatePropertyAll(0),
                                        padding: WidgetStatePropertyAll(
                                          EdgeInsets.zero,
                                        ),
                                      ),
                                      builder: (context, controller, _) =>
                                          FilledButton(
                                            style: FilledButton.styleFrom(
                                              backgroundColor: YeKe.dusky,
                                              elevation: 4,
                                              shadowColor: Colors.black54,
                                              padding: EdgeInsets.symmetric(
                                                horizontal: 12,
                                                vertical: 2,
                                              ),
                                            ),
                                            onPressed: () => controller.isOpen
                                                ? controller.close()
                                                : controller.open(),
                                            child: Row(
                                              mainAxisSize: MainAxisSize.min,
                                              children: [
                                                Text(
                                                  _city.label,
                                                  style: const TextStyle(
                                                    fontSize: 13,
                                                  ),
                                                ),
                                                const SizedBox(width: 10),
                                                CountryFlag.fromCountryCode(
                                                  _city.countryCode,
                                                  theme: const ImageTheme(
                                                    width: 20,
                                                    height: 14,
                                                  ),
                                                ),
                                              ],
                                            ),
                                          ),
                                      menuChildren: [
                                        BackdropFilter(
                                          filter: ImageFilter.blur(
                                            sigmaX: 12,
                                            sigmaY: 4,
                                          ),
                                          child: ColoredBox(
                                            color: YeKe.night.withValues(
                                              alpha: 0.5,
                                            ),
                                            child: Padding(
                                              padding:
                                                  const EdgeInsets.symmetric(
                                                    vertical: 8,
                                                  ),
                                              child: Directionality(
                                                textDirection:
                                                    TextDirection.ltr,
                                                child: Column(
                                                  mainAxisSize:
                                                      MainAxisSize.min,
                                                  crossAxisAlignment:
                                                      CrossAxisAlignment
                                                          .stretch,
                                                  children: [
                                                    for (final city
                                                        in _City.values)
                                                      MenuItemButton(
                                                        onPressed: () =>
                                                            _jumpTo(city),
                                                        trailingIcon:
                                                            CountryFlag.fromCountryCode(
                                                              city.countryCode,
                                                              theme:
                                                                  const ImageTheme(
                                                                    width: 22,
                                                                    height: 14,
                                                                  ),
                                                            ),
                                                        child: Text(city.label),
                                                      ),
                                                  ],
                                                ),
                                              ),
                                            ),
                                          ),
                                        ),
                                      ],
                                    ),
                                  ),
                                ),
                              ),
                              const SizedBox(width: 8),
                              IconButton.filled(
                                onPressed: _currentLoc == null
                                    ? null
                                    : _centerOnMe,
                                style: IconButton.styleFrom(
                                  backgroundColor: YeKe.dusky,
                                ),
                                icon: const Icon(
                                  Icons.location_pin,
                                  color: YeKe.door,
                                  size: 18,
                                ),
                              ),
                            ],
                          ),
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

class _PulseDot extends StatefulWidget {
  const _PulseDot();

  @override
  State<_PulseDot> createState() => _PulseDotState();
}

class _PulseDotState extends State<_PulseDot>
    with SingleTickerProviderStateMixin {
  late final _pulse = AnimationController(
    vsync: this,
    duration: const Duration(milliseconds: 2600),
  )..repeat();

  @override
  void dispose() {
    _pulse.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: _pulse,
      child: DecoratedBox(
        decoration: BoxDecoration(
          color: YeKe.stand,
          shape: BoxShape.circle,
          border: Border.all(color: YeKe.night, width: 3),
          boxShadow: const [BoxShadow(color: Colors.black26, blurRadius: 4)],
        ),
      ),
      builder: (context, dot) {
        final t = const Interval(
          0,
          1600 / 2600,
          curve: Curves.easeIn,
        ).transform(_pulse.value);
        return Stack(
          alignment: Alignment.center,
          children: [
            Transform.scale(
              scale: t,
              child: DecoratedBox(
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  color: YeKe.road.withValues(alpha: 0.5 * (1 - t)),
                ),
                child: const SizedBox.expand(),
              ),
            ),
            SizedBox(width: 20, height: 20, child: dot),
          ],
        );
      },
    );
  }
}
