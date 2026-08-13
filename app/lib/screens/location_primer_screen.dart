import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:geolocator/geolocator.dart';

import '../theme/ye_ke.dart';

/// The pre-permission page: says WHY before the OS popup asks. Location is a
/// routing input — which state you're in decides which authority's powers
/// apply — so this page earns the native prompt instead of ambushing the
/// user with it on frame one.
class LocationPrimerScreen extends StatefulWidget {
  final VoidCallback onFinished;

  const LocationPrimerScreen({super.key, required this.onFinished});

  @override
  State<LocationPrimerScreen> createState() => _LocationPrimerScreenState();
}

class _LocationPrimerScreenState extends State<LocationPrimerScreen> {
  bool _busy = false;

  Future<void> _allow() async {
    if (_busy) return;
    setState(() => _busy = true);

    HapticFeedback.lightImpact();
    try {
      await Geolocator.requestPermission(); // the native popup
    } catch (_) {
      // No location on this platform — nothing to ask for.
    }
    widget.onFinished();
  }

  void _skip() {
    HapticFeedback.selectionClick();
    widget.onFinished();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Container(
        decoration: const BoxDecoration(
          gradient: LinearGradient(
            begin: Alignment.topCenter,
            end: Alignment.bottomCenter,
            colors: [YeKe.night, YeKe.bg1],
          ),
        ),
        child: SafeArea(
          child: Padding(
            padding: const EdgeInsets.fromLTRB(24, 24, 24, 32),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Spacer(),
                Image.asset('assets/images/location-primer.webp'),
                Container(
                  height: 56,
                  width: 56,
                  decoration: BoxDecoration(
                    color: YeKe.road.withValues(alpha: 0.16),
                    borderRadius: BorderRadius.circular(16),
                    border: Border.all(color: YeKe.stroke),
                  ),
                  child: const Icon(
                    Icons.place,
                    color: YeKe.detained,
                    size: 28,
                  ),
                ),
                const SizedBox(height: 20),
                const Text(
                  'Get alerts of\nnearby police activity',
                  style: TextStyle(
                    color: YeKe.text,
                    fontSize: 30,
                    fontWeight: FontWeight.w800,
                    letterSpacing: -0.4,
                    height: 1.1,
                  ),
                ),
                const SizedBox(height: 14),
                const Text(
                  'Witness an abuse of power?\n'
                  "Use the livestream feature when you're in trouble.\n",
                  style: TextStyle(color: YeKe.sub, fontSize: 15, height: 1.45),
                ),
                const Text(
                  'Ye Ke does not store your location ever.',
                  style: TextStyle(
                    color: YeKe.text,
                    fontSize: 15,
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const Spacer(),
                SizedBox(
                  width: double.infinity,
                  height: 52,
                  child: FilledButton(
                    onPressed: _busy ? null : _allow,
                    style: FilledButton.styleFrom(
                      backgroundColor: YeKe.detained,
                      foregroundColor: Colors.white,
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(14),
                      ),
                    ),
                    child: const Text(
                      'Turn on location',
                      style: TextStyle(
                        fontSize: 16,
                        fontWeight: FontWeight.w800,
                      ),
                    ),
                  ),
                ),
                const SizedBox(height: 8),
                SizedBox(
                  width: double.infinity,
                  height: 44,
                  child: TextButton(
                    onPressed: _busy ? null : _skip,
                    child: const Text(
                      'Not now',
                      style: TextStyle(
                        color: YeKe.sub,
                        fontSize: 15,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
