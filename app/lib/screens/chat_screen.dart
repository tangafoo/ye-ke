import 'dart:ui';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import '../theme/ye_ke.dart';

import '../screens/ask_screen.dart';

class ChatScreen extends StatefulWidget {
  const ChatScreen({super.key});

  @override
  State<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends State<ChatScreen> {
  void _ask(String question) {
    Navigator.of(
      context,
    ).push(MaterialPageRoute(builder: (_) => AskScreen(question: question)));
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      color: YeKe.night,
      child: SafeArea(
        child: Stack(
          children: [
            Align(
              alignment: Alignment.bottomCenter,
              child: Padding(
                padding: const EdgeInsets.fromLTRB(16, 0, 16, 45),
                child: _SearchBar(hint: _placeholder, onSubmit: _ask),
              ),
            ),

            DraggableScrollableSheet(
              minChildSize: 0.02,
              initialChildSize: 0.05,
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
                        color: YeKe.paper.withValues(alpha: 0.4),
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
                                    fontSize: 20,
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

bool _bm = false;

String get _placeholder =>
    _bm ? 'Adakah polis mengganggu anda ?' : 'Are the cops bothering you ?';

class _SearchBar extends StatefulWidget {
  final String hint;
  final ValueChanged<String> onSubmit;

  const _SearchBar({required this.hint, required this.onSubmit});

  @override
  State<_SearchBar> createState() => _SearchBarState();
}

class _SearchBarState extends State<_SearchBar> {
  final _controller = TextEditingController();

  void _submit() {
    final q = _controller.text.trim();
    if (q.isEmpty) return;

    HapticFeedback.lightImpact();
    widget.onSubmit(q);
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        Image.asset('assets/images/chat-descriptor-pic.webp', width: 150),
        const SizedBox(height: 16),
        Row(
          children: [
            Expanded(
              child: TextField(
                controller: _controller,
                textInputAction: TextInputAction.search,
                onSubmitted: (_) => _submit(),
                style: const TextStyle(color: YeKe.night, fontSize: 15),
                cursorColor: YeKe.detained,
                decoration: InputDecoration(
                  border: OutlineInputBorder(
                    borderRadius: BorderRadius.zero,
                    borderSide: BorderSide.none,
                  ),
                  filled: true,
                  fillColor: YeKe.bg0,
                  hintText: widget.hint,
                  hintStyle: const TextStyle(color: YeKe.leather, fontSize: 15),
                ),
              ),
            ),
            Container(
              height: 50,
              width: 60,
              decoration: const BoxDecoration(color: YeKe.detained),
              child: Container(
                decoration: BoxDecoration(color: YeKe.detained),
                alignment: Alignment.center,
                child: const Text(
                  "Ask",
                  style: TextStyle(fontWeight: FontWeight.w900),
                ),
              ),
            ),
          ],
        ),
      ],
    );
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }
}
