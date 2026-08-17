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

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Container(
        decoration: const BoxDecoration(color: YeKe.dusky),
        child: SafeArea(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Spacer(),
                Image.asset('assets/images/notifications-primer.webp'),
                const SizedBox(height: 10),
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
                const Text(
                  'Get alerts on\nnearby abuses of power',
                  style: TextStyle(
                    color: YeKe.text,
                    fontSize: 28,
                    fontWeight: FontWeight.w800,
                    letterSpacing: -0.4,
                    height: 1.1,
                  ),
                ),
                const SizedBox(height: 16),
                const Text(
                  'Witness something fishy ?\n'
                  "Use the livestream feature to report it.\n"
                  'Notification-enabled users will receive a request to tune-in',
                  style: TextStyle(
                    color: YeKe.sub,
                    letterSpacing: 1.1,
                    fontSize: 14,
                    height: 1.5,
                  ),
                ),
                const SizedBox(height: 20),
                const Text(
                  'No marketing spam. Hopefully ever.',
                  style: TextStyle(
                    color: YeKe.text,
                    fontSize: 15,
                    letterSpacing: 1.3,
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const Spacer(),
                FilledButton(
                  onPressed: _busy ? null : _allow,
                  style: FilledButton.styleFrom(
                    backgroundColor: YeKe.detained,
                    foregroundColor: YeKe.bg0,
                    minimumSize: const Size.fromHeight(52),
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
                  width: double.infinity,
                  height: 44,
                  child: TextButton(
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
                const Spacer(),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
