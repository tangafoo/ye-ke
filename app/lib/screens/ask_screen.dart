import 'dart:async';

import '../widgets/ask_bar.dart';
import 'package:flutter/material.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';

import '../api/ask_client.dart';
import '../models/ask.dart';

import '../theme/ye_ke.dart';
import '../config.dart';

class AskScreen extends StatefulWidget {
  final String question;
  const AskScreen({super.key, required this.question});

  @override
  State<AskScreen> createState() => _AskScreenState();
}

class _AskScreenState extends State<AskScreen> {
  final client = AskClient(baseUrl: apiBaseUrl);

  final _answer = StringBuffer();
  var _citations = <Citation>[];
  String? _error;
  var _done = false;
  var _interrupted = false;

  StreamSubscription<AskEvent>? _sub;

  @override
  void initState() {
    super.initState();

    _sub = client
        .ask(widget.question)
        .listen(
          _onEvent,
          onError: (error) {
            setState(() {
              _error = error.toString();
            });
          },
          onDone: () {
            setState(() {
              _done = true;
            });
          },
        );
  }

  void _onEvent(AskEvent e) {
    setState(() {
      switch (e) {
        case CitationsEvent(:final citations):
          _citations = citations;
        case DeltaEvent(:final text):
          _answer.write(text);
        case ErrorEvent(:final message):
          _error = message;
        case DoneEvent(:final interrupted):
          _done = true;
          _interrupted = interrupted;
      }
    });
  }

  @override
  void dispose() {
    _sub?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: YeKe.bg0,
      body: Container(
        decoration: const BoxDecoration(
          gradient: LinearGradient(
            begin: Alignment.topCenter,
            end: Alignment.bottomCenter,
            colors: [YeKe.night, YeKe.leather],
          ),
        ),
        child: SafeArea(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 6, 16, 0),
                child: Row(
                  children: [
                    IconButton(
                      onPressed: () => Navigator.of(context).pop(),
                      icon: const Icon(Icons.arrow_back, color: YeKe.sub),
                    ),
                    const SizedBox(width: 4),
                    Image.asset('assets/images/logo-white.webp', height: 40),
                  ],
                ),
              ),
              const SizedBox(height: 4),
              Expanded(
                child: ListView(
                  reverse: false,
                  padding: const EdgeInsets.fromLTRB(20, 0, 20, 12),
                  children: [
                    if (_error != null)
                      Align(
                        alignment: Alignment.topCenter,
                        child: _ErrorBanner(message: _error!),
                      )
                    else
                      _Bubble.theirs(
                        '${_answer.toString()}${_done ? '' : ' ▍'}',
                      ),
                    const SizedBox(height: 14),
                    _Bubble.mine(widget.question),
                  ],
                ),
              ),

              Padding(
                padding: const EdgeInsets.fromLTRB(20, 7, 20, 6),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    AskBar(
                      hint: 'Follow up…',
                      onSubmit: (q) {
                        /* multi-turn later */
                      },
                      useIcon: true,
                    ),
                  ],
                ),
              ),

              if (_interrupted)
                const Padding(
                  padding: EdgeInsets.only(top: 8),
                  child: Text(
                    'answer may be cut short',
                    style: TextStyle(color: YeKe.stand, fontSize: 12),
                  ),
                ),

              if (_citations.isNotEmpty) ...[
                const SizedBox(height: 8),
                Padding(
                  padding: const EdgeInsets.fromLTRB(20, 0, 20, 0),
                  child: SizedBox(
                    height: 36,
                    child: ListView.separated(
                      scrollDirection: Axis.horizontal,
                      itemCount: _citations.length,
                      separatorBuilder: (_, _) => const SizedBox(width: 8),
                      itemBuilder: (context, i) {
                        final c = _citations[i];
                        return Container(
                          padding: const EdgeInsets.symmetric(horizontal: 12),
                          alignment: Alignment.center,
                          decoration: BoxDecoration(
                            color: c.related
                                ? YeKe.glow.withAlpha(50)
                                : YeKe.stand.withAlpha(50),
                            borderRadius: BorderRadius.circular(16),
                            border: Border.all(
                              color: YeKe.detained,
                              width: 1.8,
                            ),
                          ),
                          child: Text(
                            c.label,
                            style: const TextStyle(
                              color: YeKe.text,
                              fontSize: 12,
                              fontWeight: FontWeight.w600,
                            ),
                          ),
                        );
                      },
                    ),
                  ),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

class _ErrorBanner extends StatelessWidget {
  final String message;
  const _ErrorBanner({required this.message});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: YeKe.detained.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: YeKe.detained.withValues(alpha: 0.4)),
      ),
      child: Text(
        message,
        style: const TextStyle(color: YeKe.text, fontSize: 14, height: 1.4),
      ),
    );
  }
}

class _Bubble extends StatelessWidget {
  final String text;
  final bool mine;

  const _Bubble.mine(this.text) : mine = true;
  const _Bubble.theirs(this.text) : mine = false;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Align(
          alignment: mine ? Alignment.centerRight : Alignment.centerLeft,
          child: ConstrainedBox(
            constraints: BoxConstraints(
              maxWidth: MediaQuery.sizeOf(context).width * 0.78,
            ),
            child: mine
                ? Container(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 14,
                      vertical: 10,
                    ),
                    decoration: BoxDecoration(
                      boxShadow: [
                        const BoxShadow(
                          offset: Offset(7, 7),
                          color: YeKe.bg1,
                          blurRadius: 3,
                        ),
                      ],
                      color: YeKe.detained,
                      border: null,
                      borderRadius: BorderRadius.only(
                        topLeft: Radius.circular(10),
                        topRight: Radius.circular(18),
                        bottomLeft: Radius.circular(18),
                        bottomRight: Radius.circular(0),
                      ),
                    ),
                    child: Text(
                      text,
                      style: const TextStyle(color: YeKe.bg0, fontSize: 15),
                    ),
                  )
                : Container(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 14,
                      vertical: 10,
                    ),
                    decoration: BoxDecoration(
                      boxShadow: [
                        const BoxShadow(
                          offset: Offset(-5, 7),
                          color: YeKe.night,
                          blurStyle: BlurStyle.inner,
                          blurRadius: 7,
                        ),
                      ],
                      color: YeKe.paper,
                      border: Border.all(color: YeKe.stroke),
                      borderRadius: BorderRadius.only(
                        topLeft: Radius.circular(18),
                        topRight: Radius.circular(10),
                        bottomLeft: Radius.circular(0),
                        bottomRight: Radius.circular(18),
                      ),
                    ),
                    child: MarkdownBody(
                      data: text,
                      styleSheet: MarkdownStyleSheet(
                        p: const TextStyle(
                          color: YeKe.night,
                          fontSize: 15,
                          fontFamily: 'Faculty',
                        ),
                        strong: const TextStyle(
                          color: YeKe.night,
                          fontSize: 16,
                          fontFamily: 'Faculty',
                          fontWeight: FontWeight.bold,
                        ),
                        em: const TextStyle(
                          color: YeKe.leather,
                          fontStyle: FontStyle.italic,
                        ),
                        h2: const TextStyle(
                          color: YeKe.detained,
                          fontSize: 19,
                          letterSpacing: -0.3,
                        ),
                        listBullet: const TextStyle(
                          color: YeKe.leather,
                          fontSize: 16,
                          fontWeight: FontWeight.bold,
                        ),
                        blockSpacing: 10,
                      ),
                    ),
                  ),
          ),
        ),
      ],
    );
  }
}
