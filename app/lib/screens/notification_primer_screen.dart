import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:permission_handler/permission_handler.dart';

import '../theme/ye_ke.dart';

class NotificationPrimerScreen extends StatefulWidget {
  final VoidCallback onFinished;

  const NotificationPrimerScreen({super.key, required this.onFinished});

  @override
  State<NotificationPrimerScreen> createState() =>
      _NotificationPrimerScreenState();
}

class _NotificationPrimerScreenState extends State<NotificationPrimerScreen> {
  bool _busy = false;

  Future<void> _allow() async {
    if (_busy) return;
    setState(() => _busy = true);

    HapticFeedback.lightImpact();
    try {
      await Permission.notification.request();
    } catch (_) {}

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
                    "Use the livestream feature to report it.\n"
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
                  FilledButton(
                    onPressed: _busy ? null : _allow,
                    style: FilledButton.styleFrom(
                      backgroundColor: YeKe.rotom,
                      foregroundColor: YeKe.white,
                      alignment: desktop
                          ? Alignment.centerLeft
                          : Alignment.center,
                      minimumSize: desktop
                          ? const Size(_wideButtonWidth, 52)
                          : const Size.fromHeight(52),
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(12),
                      ),
                      textStyle: const TextStyle(
                        letterSpacing: 1.1,
                        fontFamily: 'Chelsea',
                        fontSize: 16,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    child: Text('Turn on Notifications'),
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
