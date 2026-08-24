class Ping {
  final String id;
  final String kind;
  final double lat;
  final double lng;
  final DateTime createdAt;
  final bool mine;

  const Ping({
    required this.id,
    required this.kind,
    required this.lat,
    required this.lng,
    required this.createdAt,
    required this.mine,
  });

  factory Ping.fromJson(Map<String, dynamic> json) => Ping(
    id: json['id'] as String? ?? '',
    kind: json['kind'] as String? ?? '',
    lat: (json['lat'] as num?)?.toDouble() ?? 0,
    lng: (json['lng'] as num?)?.toDouble() ?? 0,
    createdAt:
        DateTime.tryParse(json['created_at'] as String? ?? '') ??
        DateTime.now(),
    mine: json['mine'] as bool? ?? false,
  );
}
