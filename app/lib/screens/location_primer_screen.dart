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

  static const _wideBreakpoint = 840.0;
  static const _wideButtonWidth = 320.0;

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
            child: LayoutBuilder(
              builder: (context, constraints) {
                final desktop = constraints.maxWidth >= _wideBreakpoint;
                final textAlign = desktop ? TextAlign.end : TextAlign.start;

                final closingMessage = Text(
                  'YeKe never stores your location, ever.',
                  textAlign: textAlign,
                  style: TextStyle(
                    color: YeKe.text,
                    fontSize: 15,
                    letterSpacing: 1.1,
                    fontWeight: FontWeight.w700,
                  ),
                );

                final content = <Widget>[
                  Flexible(
                    flex: 3,
                    child: Image.asset(
                      'assets/images/location-primer.webp',
                      fit: BoxFit.contain,
                    ),
                  ),
                  if (!desktop) ...[
                    Container(
                      height: 56,
                      width: 56,
                      decoration: BoxDecoration(
                        color: YeKe.white.withValues(alpha: 0.16),
                        borderRadius: BorderRadius.circular(16),
                        border: Border.all(color: YeKe.road),
                      ),
                      child: const Icon(
                        Icons.place,
                        color: YeKe.rotom,
                        size: 34,
                      ),
                    ),
                    const SizedBox(height: 16),
                  ],
                  Text(
                    'Check areas with recent police activity',
                    textAlign: textAlign,
                    style: TextStyle(
                      color: YeKe.text,
                      fontSize: 30,
                      fontWeight: FontWeight.w800,
                      letterSpacing: -0.4,
                      height: 1.2,
                    ),
                  ),
                  const SizedBox(height: 16),
                  Text(
                    "Use the 'Ping' feature to quickly report\npolice presence.\n"
                    "'Historic View' to check hotspot areas & notable incidents",
                    textAlign: textAlign,
                    style: TextStyle(
                      color: YeKe.sub,
                      letterSpacing: 1.2,
                      fontSize: 14,
                      height: 1.50,
                    ),
                  ),
                  if (!desktop) ...[const SizedBox(height: 24), closingMessage],
                ];

                final buttons = <Widget>[
                  if (desktop) ...[
                    Container(
                      height: 56,
                      width: 56,
                      decoration: BoxDecoration(
                        color: YeKe.white.withValues(alpha: 0.16),
                        borderRadius: BorderRadius.circular(16),
                        border: Border.all(color: YeKe.road),
                      ),
                      child: const Icon(
                        Icons.place,
                        color: YeKe.rotom,
                        size: 34,
                      ),
                    ),
                    const SizedBox(height: 12),
                    closingMessage,
                    const SizedBox(height: 20),
                  ],
                  SizedBox(
                    width: desktop ? _wideButtonWidth : double.infinity,
                    height: 52,
                    child: FilledButton(
                      onPressed: _busy ? null : _allow,
                      style: FilledButton.styleFrom(
                        backgroundColor: YeKe.rotom,
                        foregroundColor: Colors.white,
                        alignment: desktop
                            ? Alignment.centerLeft
                            : Alignment.center,
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
                    width: desktop ? _wideButtonWidth : double.infinity,
                    height: 44,
                    child: TextButton(
                      style: TextButton.styleFrom(
                        alignment: desktop
                            ? Alignment.centerLeft
                            : Alignment.center,
                        padding: const EdgeInsets.only(left: 24),
                      ),
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
                ];

                if (!desktop) {
                  return Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Spacer(),
                      ...content,
                      const Spacer(),
                      ...buttons,
                      const Spacer(),
                    ],
                  );
                }

                return Row(
                  children: [
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.end,
                        children: [const Spacer(), ...content, const Spacer()],
                      ),
                    ),
                    const SizedBox(width: 48),
                    Expanded(
                      child: Column(
                        mainAxisAlignment: MainAxisAlignment.center,
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: buttons,
                      ),
                    ),
                  ],
                );
              },
            ),
          ),
        ),
      ),
    );
  }
}
