import 'dart:convert';
import '../api/sse.dart';

class Citation {
  final String statute;
  final String statuteAbbr;
  final String actNumber;
  final String section;
  final String subsection;
  final String heading;
  final String text;
  final String sourceUrl;
  final bool related;

  const Citation({
    required this.statute,
    required this.statuteAbbr,
    required this.actNumber,
    required this.section,
    required this.subsection,
    required this.heading,
    required this.text,
    required this.sourceUrl,
    required this.related,
  });

  factory Citation.fromJson(Map<String, dynamic> json) {
    return Citation(
      statute: json['statute'] as String? ?? '',
      statuteAbbr: json['statute_abbr'] as String? ?? '',
      actNumber: json['act_number'] as String? ?? '',
      section: json['section'] as String? ?? '',
      subsection: json['subsection'] as String? ?? '',
      heading: json['heading'] as String? ?? '',
      text: json['text'] as String? ?? '',
      sourceUrl: json['source_url'] as String? ?? '',
      related: json['related'] as bool? ?? false,
    );
  }
}

sealed class AskEvent {
  const AskEvent();

  static AskEvent? fromSse(SseEvent e) {
    switch (e.event) {
      case 'citations':
        final list = jsonDecode(e.data) as List<dynamic>;
        return CitationsEvent([
          for (final c in list) Citation.fromJson(c as Map<String, dynamic>),
        ]);
      case 'delta':
        return DeltaEvent((jsonDecode(e.data) as Map)['text'] as String? ?? '');
      case 'error':
        return ErrorEvent(
          (jsonDecode(e.data) as Map)['message'] as String? ?? 'unknown error',
        );
      case 'done':
        return DoneEvent(
          (jsonDecode(e.data) as Map)['interrupted'] as bool? ?? false,
        );
      default:
        return null;
    }
  }
}

class CitationsEvent extends AskEvent {
  final List<Citation> citations;
  const CitationsEvent(this.citations);
}

class DeltaEvent extends AskEvent {
  final String text;
  const DeltaEvent(this.text);
}

class ErrorEvent extends AskEvent {
  final String message;
  const ErrorEvent(this.message);
}

class DoneEvent extends AskEvent {
  final bool interrupted;
  const DoneEvent(this.interrupted);
}
