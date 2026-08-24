import 'dart:math';

import 'package:flutter/material.dart';

import '../theme/ye_ke.dart';

class RadarView extends StatelessWidget {
  const RadarView({super.key});

  @override
  Widget build(BuildContext context) {
    return RepaintBoundary(
      child: SizedBox.expand(child: CustomPaint(painter: RadarPainter())),
    );
  }
}

class RadarPainter extends CustomPainter {
  static const rings = 3;

  @override
  void paint(Canvas canvas, Size size) {
    final center = Offset(size.width / 2, size.height / 2);
    final radius = min(size.width, size.height) / 2 - 8;

    final line = Paint()
      ..color = YeKe.paint
      ..style = PaintingStyle.stroke
      ..strokeWidth = 1;

    for (var i = 1; i <= rings; i++) {
      canvas.drawCircle(center, radius * i / rings, line);
    }

    canvas.drawLine(
      center + const Offset(0, -1) * radius,
      center + const Offset(0, 1) * radius,
      line,
    );
    canvas.drawLine(
      center + const Offset(-1, 0) * radius,
      center + const Offset(1, 0) * radius,
      line,
    );

    canvas.drawCircle(center, 8, Paint()..color = YeKe.detained);

    final n = TextPainter(
      text: const TextSpan(
        text: 'N',
        style: TextStyle(color: YeKe.oddish, fontSize: 10),
      ),
      textDirection: TextDirection.ltr,
    )..layout();

    n.paint(canvas, center + Offset(-n.width / 2, -radius - n.height - 2));
  }

  @override
  bool shouldRepaint(covariant RadarPainter oldDelegate) => false;
}
