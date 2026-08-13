class SseEvent {
  final String event;
  final String data;
  const SseEvent(this.event, this.data);
}

class SseDecoder {
  String _buf = '';

  // Will return []
  List<SseEvent> feed(String chunk) {
    _buf = (_buf + chunk).replaceAll('\r\n', '\n');

    final out = <SseEvent>[];

    // chunk/_buf looks like: event: delta\ndata: {"text":"hi"}\n\nevent: citations\ndata: [{"act":"RTA 1987","section":"45B"}]\n\n
    var separator = _buf.indexOf('\n\n');

    while (separator != -1) {
      final raw = _buf.substring(0, separator);
      _buf = _buf.substring(separator + 2);

      final e = _parseBlock(raw);
      if (e != null) out.add(e);

      separator = _buf.indexOf('\n\n');
    }

    return out;
  }

  SseEvent? _parseBlock(String raw) {
    var name = 'message';
    final data = <String>[];

    for (final line in raw.split('\n')) {
      // SSE conventions - middlewares 'might' return a comment - and those start with ':'
      if (line.startsWith(':')) continue;
      if (line.startsWith('event:')) {
        name = _fieldValue(line, 'event:');
      } else if (line.startsWith('data:')) {
        data.add(_fieldValue(line, 'data:'));
      }
    }

    if (data.isEmpty) return null;
    return SseEvent(name, data.join('\n'));
  }

  String _fieldValue(String line, String field) {
    var v = line.substring(field.length);
    if (v.startsWith(' ')) v = v.substring(1);
    return v;
  }
}
