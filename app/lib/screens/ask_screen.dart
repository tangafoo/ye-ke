import 'dart:async';

import 'package:flutter/material.dart';

import '../api/ask_client.dart';
import '../models/ask.dart';

import '../theme/moth.dart';

class AskScreen extends StatefulWidget {
  final String question;
  const AskScreen({super.key, required this.question});

  @override
  State<AskScreen> createState() => _AskScreenState();
}

class _AskScreenState extends State<AskScreen> {
  final client = AskClient(baseUrl: 'http://localhost:8080');

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
    return const Placeholder();
  }
}
