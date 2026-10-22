import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:permission_handler/permission_handler.dart';

import '../theme/ye_ke.dart';

class YekePrimerScreen extends StatefulWidget {
  final VoidCallback onFinished;

  const YekePrimerScreen({super.key, required this.onFinished});

  @override
  State<YekePrimerScreen> createState() => _YekePrimerScreenState();
}

class _YekePrimerScreenState extends State<YekePrimerScreen> {
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

  static const _wideBreakpoint = 840.0;
  static const _wideButtonWidth = 320.0;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Container(
        decoration: const BoxDecoration(color: YeKe.scaley),
        child: SafeArea(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: LayoutBuilder(
              builder: (context, constraints) {
                final desktop = constraints.maxWidth >= _wideBreakpoint;
                final textAlign = desktop ? TextAlign.end : TextAlign.start;

                final closingMessage = Text(
                  'Do not use me as a defense lawyer.',
                  textAlign: textAlign,
                  style: TextStyle(
                    color: YeKe.text,
                    fontSize: 15,
                    fontWeight: FontWeight.w700,
                  ),
                );

                final content = <Widget>[
                  Flexible(
                    flex: 3,
                    child: Image.asset(
                      'assets/images/yeke-brain-primer.webp',
                      fit: BoxFit.contain,
                    ),
                  ),
                  const SizedBox(height: 10),
                  if (!desktop) ...[
                    Container(
                      height: 56,
                      width: 56,
                      decoration: BoxDecoration(
                        color: YeKe.sceptile.withValues(alpha: 0.16),
                        borderRadius: BorderRadius.circular(16),
                        border: Border.all(color: YeKe.paper),
                      ),
                      child: const Icon(
                        Icons.all_inclusive,
                        color: YeKe.oddish,
                        size: 28,
                      ),
                    ),
                    const SizedBox(height: 20),
                  ],
                  Text(
                    "YeKe helps stand your ground",
                    textAlign: textAlign,
                    style: TextStyle(
                      color: YeKe.text,
                      fontSize: 28,
                      fontWeight: FontWeight.w800,
                      letterSpacing: -0.4,
                      height: 1.1,
                    ),
                  ),
                  const SizedBox(height: 20),
                  Text(
                    "Use the 'Chat' feature in any law situation\n"
                    "'Urgent Mode' and 'Voice Mode' available\n"
                    "when time-sensitive.\n"
                    "Retrieval from over 1,000+ acts.\n"
                    "Draft legal action forms for offending officers",
                    textAlign: textAlign,
                    style: TextStyle(color: YeKe.sub, fontSize: 14, height: 1.5),
                  ),
                  if (!desktop) ...[const SizedBox(height: 20), closingMessage],
                ];

                final buttons = <Widget>[
                  if (desktop) ...[
                    Container(
                      height: 56,
                      width: 56,
                      decoration: BoxDecoration(
                        color: YeKe.sceptile.withValues(alpha: 0.16),
                        borderRadius: BorderRadius.circular(16),
                        border: Border.all(color: YeKe.paper),
                      ),
                      child: const Icon(
                        Icons.all_inclusive,
                        color: YeKe.oddish,
                        size: 28,
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
                    child: Text('Finish'),
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
