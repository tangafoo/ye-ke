import 'dart:convert';

import 'package:http/http.dart' as http;
import '../models/ask.dart';
import 'sse.dart';

class AskClient {
  final String baseUrl;
  final http.Client _http;

  AskClient({required this.baseUrl, http.Client? client})
    : _http = client ?? http.Client();

  Stream<AskEvent> ask(String question, {String lang = 'en'}) async* {
    final req = http.Request('POST', Uri.parse('$baseUrl/ask'));

    req.headers['Content-Type'] = 'application/json';
    req.body = jsonEncode({'question': question, 'lang': lang, 'stream': true});

    final http.StreamedResponse res;

    try {
      res = await _http.send(req);
    } on http.ClientException catch (e) {
      yield ErrorEvent(
        'cannot reach Ye Ke atm - check your connection: ${e.message}',
      );
      return;
    }

    if (res.statusCode != 200) {
      final msg = await res.stream.bytesToString();

      yield ErrorEvent(msg.trim());
      return;
    }

    final contentType = res.headers['content-type'] ?? '';

    // If res sent as whole block instead of stream - handle & display all
    if (!contentType.contains('text/event-stream')) {
      final body =
          jsonDecode(await res.stream.bytesToString()) as Map<String, dynamic>;

      final list = body['citations'] as List<dynamic>? ?? [];

      yield CitationsEvent([
        for (final c in list) Citation.fromJson(c as Map<String, dynamic>),
      ]);
      yield DeltaEvent(body['answer'] as String? ?? '');
      yield DoneEvent(false);
      return;
    }

    final decoder = SseDecoder();

    await for (final chunk in res.stream.transform(utf8.decoder)) {
      for (final se in decoder.feed(chunk)) {
        final event = AskEvent.fromSse(se);

        if (event != null) {
          yield event;
        }
      }
    }
  }
}
