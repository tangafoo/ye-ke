import 'dart:ui';

import 'package:flutter/material.dart';
import '../theme/ye_ke.dart';

import '../screens/ask_screen.dart';
import '../widgets/ask_bar.dart';

class ChatScreen extends StatefulWidget {
  const ChatScreen({super.key});

  @override
  State<ChatScreen> createState() => _ChatScreenState();
}

bool _bm = false;

String get _placeholder =>
    _bm ? 'Adakah polis mengganggu anda ?' : 'Are the cops bothering you ?';

class _ChatScreenState extends State<ChatScreen> {
  void _ask(String question) {
    Navigator.of(
      context,
    ).push(MaterialPageRoute(builder: (_) => AskScreen(question: question)));
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: const BoxDecoration(
        gradient: LinearGradient(
          begin: Alignment.topCenter,
          end: Alignment.bottomCenter,
          colors: [YeKe.night, YeKe.bg1],
        ),
      ),
      child: SafeArea(
        child: Stack(
          children: [
            Align(
              alignment: Alignment.bottomCenter,
              child: Padding(
                padding: const EdgeInsets.fromLTRB(16, 0, 16, 45),
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.end,
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Image.asset(
                      'assets/images/chat-descriptor-pic.webp',
                      width: 150,
                    ),
                    const SizedBox(height: 8),
                    AskBar(hint: _placeholder, onSubmit: _ask),
                  ],
                ),
              ),
            ),

            DraggableScrollableSheet(
              minChildSize: 0.02,
              initialChildSize: 0.03,
              maxChildSize: 0.30,
              snap: true,
              builder: (context, scrollController) {
                return ClipRRect(
                  borderRadius: const BorderRadius.vertical(
                    top: Radius.circular(12),
                  ),
                  child: BackdropFilter(
                    filter: ImageFilter.blur(sigmaX: 22, sigmaY: 22),
                    child: Container(
                      decoration: BoxDecoration(
                        color: YeKe.leather.withValues(alpha: 0.6),
                        border: const Border(
                          top: BorderSide(color: Color(0x33FFFFFF)),
                        ),
                      ),
                      child: ListView(
                        controller: scrollController,
                        children: [
                          Padding(
                            padding: const EdgeInsets.all(24),
                            child: Align(
                              alignment: Alignment.centerRight,
                              child: Container(
                                padding: const EdgeInsets.all(12),
                                decoration: BoxDecoration(
                                  boxShadow: [
                                    BoxShadow(
                                      blurRadius: 6,
                                      offset: Offset(0, 6),
                                      blurStyle: BlurStyle.inner,
                                    ),
                                  ],
                                  borderRadius: BorderRadius.circular(12),
                                  color: YeKe.stand,
                                ),
                                child: Text(
                                  "Urgent Mode",
                                  style: TextStyle(
                                    fontSize: 18,
                                    color: Color(0xFF000000),
                                    fontWeight: FontWeight.w800,
                                  ),
                                ),
                              ),
                            ),
                          ),
                          Padding(
                            padding: const EdgeInsets.all(24),
                            child: Align(
                              alignment: Alignment.centerRight,
                              child: Container(
                                padding: const EdgeInsets.all(12),
                                decoration: BoxDecoration(
                                  boxShadow: [
                                    BoxShadow(
                                      blurRadius: 6,
                                      offset: Offset(0, 6),
                                      blurStyle: BlurStyle.inner,
                                    ),
                                  ],
                                  borderRadius: BorderRadius.circular(12),
                                  color: YeKe.detained,
                                ),
                                child: Text(
                                  "Voice Mode",
                                  style: TextStyle(
                                    fontSize: 18,
                                    color: Color(0xFF000000),
                                    fontWeight: FontWeight.w800,
                                  ),
                                ),
                              ),
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
                );
              },
            ),
          ],
        ),
      ),
    );
  }
}
