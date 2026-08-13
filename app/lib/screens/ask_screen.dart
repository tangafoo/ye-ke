import 'dart:async';

import 'package:flutter/material.dart';

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
          child: Padding(
            padding: const EdgeInsets.fromLTRB(20, 12, 20, 20),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    IconButton(
                      onPressed: () => Navigator.of(context).pop(),
                      icon: const Icon(Icons.arrow_back, color: YeKe.sub),
                    ),
                    const SizedBox(width: 4),
                    Image.asset('assets/images/logo-white.webp', height: 40),
                  ],
                ),
                const SizedBox(height: 12),

                Expanded(
                  child: ListView(
                    reverse: false,
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

                if (_interrupted)
                  const Padding(
                    padding: EdgeInsets.only(top: 8),
                    child: Text(
                      'answer may be cut short',
                      style: TextStyle(color: YeKe.stand, fontSize: 12),
                    ),
                  ),

                if (_citations.isNotEmpty) ...[
                  const SizedBox(height: 14),
                  SizedBox(
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
                            color: YeKe.glass,
                            borderRadius: BorderRadius.circular(18),
                            border: Border.all(
                              color: c.related ? YeKe.stroke : YeKe.glow,
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
                ],
              ],
            ),
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
    return Align(
      alignment: mine ? Alignment.centerRight : Alignment.centerLeft,
      child: ConstrainedBox(
        constraints: BoxConstraints(
          maxWidth: MediaQuery.sizeOf(context).width * 0.78,
        ),
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
          decoration: BoxDecoration(
            color: mine ? YeKe.detained : YeKe.glass,
            border: mine ? null : Border.all(color: YeKe.stroke),
            borderRadius: BorderRadius.only(
              topLeft: Radius.circular(mine ? 10 : 18),
              topRight: Radius.circular(mine ? 18 : 10),
              bottomLeft: Radius.circular(mine ? 18 : 0),
              bottomRight: Radius.circular(mine ? 0 : 18),
            ),
          ),
          child: Text(
            text,
            style: TextStyle(
              color: mine ? YeKe.bg0 : YeKe.text,
              fontSize: 15,
              fontWeight: mine ? FontWeight.w700 : FontWeight.w400,
              fontFamily: mine ? null : 'Faculty',
              height: 1.5,
            ),
          ),
        ),
      ),
    );
  }
}
