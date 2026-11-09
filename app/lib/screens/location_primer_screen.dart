import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:geolocator/geolocator.dart';

import '../theme/ye_ke.dart';

enum _PrimerState { unasked, granted, denied }

/// The pre-permission page: says WHY before the OS popup asks. Location is a
/// routing input — which state you're in decides which authority's powers
/// apply — so this page earns the native prompt instead of ambushing the
/// user with it on frame one.
class LocationPrimerScreen extends StatefulWidget {
  final VoidCallback onFinished;
  final bool permissionGiven;

  const LocationPrimerScreen({
    super.key,
    required this.onFinished,
    this.permissionGiven = false,
  });

  @override
  State<LocationPrimerScreen> createState() => _LocationPrimerScreenState();
}

class _LocationPrimerScreenState extends State<LocationPrimerScreen>
    with AutomaticKeepAliveClientMixin {
  bool _busy = false;
  _PrimerState _state = _PrimerState.unasked;

  static const _advanceDelay = Duration(milliseconds: 1600);

  @override
  bool get wantKeepAlive => true;

  @override
  void initState() {
    super.initState();
    if (widget.permissionGiven) _state = _PrimerState.granted;
  }

  _PrimerState _fromLocation(LocationPermission p) => switch (p) {
    LocationPermission.whileInUse ||
    LocationPermission.always => _PrimerState.granted,
    LocationPermission.denied ||
    LocationPermission.deniedForever ||
    LocationPermission.unableToDetermine => _PrimerState.denied,
  };

  Future<void> _allow() async {
    if (_busy) return;
    setState(() => _busy = true);

    HapticFeedback.lightImpact();
    var result = _PrimerState.denied;

    try {
      result = _fromLocation(await Geolocator.requestPermission());
    } catch (_) {
      // No location on this platform — nothing to ask for.
    }

    if (!mounted) return;
    setState(() => _state = result);

    await Future.delayed(_advanceDelay);
    if (!mounted) return;
    widget.onFinished();
  }

  void _skip() {
    if (_busy) return;

    HapticFeedback.selectionClick();
    widget.onFinished();
  }

  static const _wideBreakpoint = 840.0;
  static const _wideButtonWidth = 320.0;

  @override
  Widget build(BuildContext context) {
    super.build(context);

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
                    "'Radar View' to check hot spots & notable incidents",
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

                final responsiveButtonStyling = FilledButton.styleFrom(
                  alignment: desktop ? Alignment.centerLeft : Alignment.center,
                  minimumSize: desktop
                      ? const Size(_wideButtonWidth, 52)
                      : const Size.fromHeight(52),
                );

                final notGrantedActions = Column(
                  key: const ValueKey(_PrimerState.unasked),
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    FilledButton(
                      onPressed: _allow,
                      style: responsiveButtonStyling,
                      child: _busy
                          ? const SizedBox(
                              height: 20,
                              width: 20,
                              child: CircularProgressIndicator(
                                strokeWidth: 2,
                                color: YeKe.white,
                              ),
                            )
                          : const Text('Turn on location'),
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
                  ],
                );

                // Permission given is not a $state. Given on app start ups. GB has 2 features catching close app / halfway onboarding behaviour
                final grantedButton = FilledButton(
                  onPressed: widget.permissionGiven ? widget.onFinished : null,
                  style: responsiveButtonStyling,
                  child: const Text('Location is ON'),
                );

                final grantedActions = Row(
                  key: const ValueKey(_PrimerState.granted),
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    desktop ? grantedButton : Expanded(child: grantedButton),
                    const SizedBox(width: 12),
                    Container(
                      height: 40,
                      width: 40,
                      alignment: Alignment.center,
                      decoration: BoxDecoration(
                        border: Border.all(
                          color: YeKe.oddish.withValues(alpha: 0.5),
                          width: 2,
                        ),
                        borderRadius: BorderRadius.circular(10),
                      ),
                      child: Transform.rotate(
                        angle: 0.2,
                        child: const Icon(
                          Icons.check_box_outlined,
                          color: YeKe.oddish,
                          size: 35,
                        ),
                      ),
                    ),
                  ],
                );

                final deniedActions = const Text(
                  'Location turned OFF.\nYou can change this later in Settings.',
                  key: ValueKey(_PrimerState.denied),
                  style: TextStyle(
                    color: YeKe.cement,
                    fontWeight: FontWeight.w500,
                    fontSize: 15,
                  ),
                );

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
                  AnimatedSwitcher(
                    duration: const Duration(milliseconds: 250),
                    layoutBuilder: (current, previous) => Stack(
                      alignment: AlignmentDirectional.centerStart,
                      children: [...previous, ?current],
                    ),
                    child: switch (_state) {
                      _PrimerState.unasked => notGrantedActions,
                      _PrimerState.granted => grantedActions,
                      _PrimerState.denied => deniedActions,
                    },
                  ),
                  if (_state == _PrimerState.granted) ...[
                    const SizedBox(height: 12),
                    Text(
                      'You can change this later in Settings.',
                      style: TextStyle(
                        fontFamily: 'Faculty',
                        color: YeKe.white.withValues(alpha: 0.5),
                      ),
                    ),
                  ],
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
