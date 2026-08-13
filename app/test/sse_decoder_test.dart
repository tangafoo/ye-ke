import 'package:flutter_test/flutter_test.dart';
import 'package:byakugan/api/sse.dart';

void main() {
  test('one complete event', () {
    final d = SseDecoder();
    final events = d.feed('event: delta\ndata: {"text":"hi"}\n\n');

    expect(events, hasLength(1));
    expect(events[0].event, 'delta');
    expect(events[0].data, '{"text":"hi"}');
  });

  test('event split across two chunks', () {
    final d = SseDecoder();
    expect(d.feed('event: delta\ndata: {"tex'), isEmpty);

    final events = d.feed('t":"hi"}\n\n');
    expect(events, hasLength(1));
    expect(events[0].event, 'delta');
    expect(events[0].data, '{"text":"hi"}');
  });

  test('two events in one chunk', () {
    final d = SseDecoder();
    final events = d.feed(
      'event: citations\ndata: []\n\n'
      'event: delta\ndata: {"text":"go"}\n\n',
    );

    expect(events, hasLength(2));
    expect(events[0].event, 'citations');
    expect(events[1].event, 'delta');
  });

  test('complete event plus trailing partial — holds the tail', () {
    final d = SseDecoder();
    final events = d.feed(
      'event: done\ndata: {"interrupted":false}\n\nevent: del',
    );

    expect(events, hasLength(1));
    expect(events[0].event, 'done');

    final more = d.feed('ta\ndata: {"text":"x"}\n\n');
    expect(more, hasLength(1));
    expect(more[0].event, 'delta');
  });

  test('delta text keeps its own leading space', () {
    final d = SseDecoder();

    final events = d.feed('event: delta\ndata:  padded\n\n');
    expect(events[0].data, ' padded');
  });
}
