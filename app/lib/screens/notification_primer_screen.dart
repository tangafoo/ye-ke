import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:permission_handler/permission_handler.dart';

import '../theme/ye_ke.dart';

enum _PrimerState { unasked, granted, denied }

class NotificationPrimerScreen extends StatefulWidget {
  final VoidCallback onFinished;
  final bool permissionGiven;

  const NotificationPrimerScreen({
    super.key,
    required this.onFinished,
    this.permissionGiven = false,
  });

  @override
  State<NotificationPrimerScreen> createState() =>
      _NotificationPrimerScreenState();
}

class _NotificationPrimerScreenState extends State<NotificationPrimerScreen>
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

  _PrimerState _fromStatus(PermissionStatus s) => switch (s) {
    PermissionStatus.granted ||
    PermissionStatus.provisional ||
    PermissionStatus.limited => _PrimerState.granted,
    PermissionStatus.denied ||
    PermissionStatus.permanentlyDenied ||
    PermissionStatus.restricted => _PrimerState.denied,
  };

  Future<void> _allow() async {
    if (_busy) return;
    setState(() => _busy = true);

    HapticFeedback.lightImpact();
    var result = _PrimerState.denied;

    try {
      result = _fromStatus(await Permission.notification.request());
    } catch (_) {}

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
        decoration: const BoxDecoration(color: YeKe.dusky),
        child: SafeArea(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: LayoutBuilder(
              builder: (context, constraints) {
                final desktop = constraints.maxWidth >= _wideBreakpoint;
                final textAlign = desktop ? TextAlign.end : TextAlign.start;

                final closingMessage = Text(
                  'No marketing spam. Hopefully ever.',
                  textAlign: textAlign,
                  style: TextStyle(
                    color: YeKe.text,
                    fontSize: 15,
                    letterSpacing: 1.3,
                    fontWeight: FontWeight.w700,
                  ),
                );

                final content = <Widget>[
                  Flexible(
                    flex: 3,
                    child: Image.asset(
                      'assets/images/notifications-primer.webp',
                      fit: BoxFit.contain,
                    ),
                  ),
                  const SizedBox(height: 10),
                  if (!desktop) ...[
                    Container(
                      height: 56,
                      width: 56,
                      alignment: Alignment.center,
                      decoration: BoxDecoration(
                        color: YeKe.road.withValues(alpha: 0.16),
                        borderRadius: BorderRadius.circular(16),
                        border: Border.all(color: YeKe.night),
                      ),
                      child: Image.asset(
                        'assets/images/camera-icon.webp',
                        height: 35,
                      ),
                    ),
                    const SizedBox(height: 20),
                  ],
                  Text(
                    'Get alerts on\nnearby abuses of power',
                    textAlign: textAlign,
                    style: TextStyle(
                      color: YeKe.text,
                      fontSize: 28,
                      fontWeight: FontWeight.w800,
                      letterSpacing: -0.4,
                      height: 1.1,
                    ),
                  ),
                  const SizedBox(height: 16),
                  Text(
                    'Witness something fishy ?\n'
                    "Use the livestream feature to snap it.\n"
                    'Notification-enabled users will receive a request to tune-in',
                    textAlign: textAlign,
                    style: TextStyle(
                      color: YeKe.sub,
                      letterSpacing: 1.1,
                      fontSize: 14,
                      height: 1.5,
                    ),
                  ),
                  if (!desktop) ...[const SizedBox(height: 20), closingMessage],
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
                          : const Text('Turn on Notifications'),
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
                        child: Text(
                          'Skip for now',
                          style: TextStyle(
                            color: YeKe.cement,
                            fontWeight: FontWeight.w600,
                            fontSize: 15,
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
                  child: const Text('Alerts are ON'),
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
                        angle: -0.2,
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
                  'Alerts turned OFF.\nYou can change this later in Settings.',
                  key: ValueKey(_PrimerState.denied),
                  style: TextStyle(
                    color: YeKe.cement,
                    fontWeight: FontWeight.w600,
                    fontSize: 15,
                  ),
                );

                final buttons = <Widget>[
                  if (desktop) ...[
                    Container(
                      height: 56,
                      width: 56,
                      alignment: Alignment.center,
                      decoration: BoxDecoration(
                        color: YeKe.road.withValues(alpha: 0.16),
                        borderRadius: BorderRadius.circular(16),
                        border: Border.all(color: YeKe.night),
                      ),
                      child: Image.asset(
                        'assets/images/camera-icon.webp',
                        height: 35,
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
