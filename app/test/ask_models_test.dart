import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:byakugan/api/sse.dart';
import 'package:byakugan/models/ask.dart';

void main() {
  test('Go res can parse into Citations', () {
    final raw = File('test/fixtures/citations.json').readAsStringSync();
    final list = jsonDecode(raw) as List<dynamic>;

    final citations = [
      for (final c in list) Citation.fromJson(c as Map<String, dynamic>),
    ];

    expect(citations, isNotEmpty);
    expect(citations.first.statuteAbbr, 'AA 1960');
    expect(citations[1].section, '8');
    expect(citations[3].subsection, '7(2)');
    expect(citations.first.sourceUrl, startsWith('http'));
    expect(citations.last.related, isTrue);
  });

  test('AskEvents propagating properly', () {
    final delta = AskEvent.fromSse(SseEvent('delta', '{"text":"hi"}'));
    expect(delta, isA<DeltaEvent>());
    expect((delta as DeltaEvent).text, 'hi');

    final err = AskEvent.fromSse(SseEvent('error', '{"message":"nope"}'));
    expect((err as ErrorEvent).message, 'nope');

    final done = AskEvent.fromSse(SseEvent('done', '{"interrupted":true}'));
    expect((done as DoneEvent).interrupted, isTrue);

    expect(AskEvent.fromSse(SseEvent('confetti', '{}')), isNull);
  });

  test('Citations get parsed to citations event', () {
    final raw = File('test/fixtures/citations.json').readAsStringSync();
    final e = AskEvent.fromSse(SseEvent('citations', raw));

    expect(e, isA<CitationsEvent>());
    expect((e as CitationsEvent).citations, isNotEmpty);
  });
}
