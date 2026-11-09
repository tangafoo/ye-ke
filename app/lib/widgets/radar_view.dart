import 'dart:math';

import 'package:flutter/material.dart';

import '../theme/ye_ke.dart';

class RadarView extends StatefulWidget {
  const RadarView({super.key});

  @override
  State<RadarView> createState() => _RadarViewState();
}

class _RadarViewState extends State<RadarView>
    with SingleTickerProviderStateMixin {
  late final _sweep = AnimationController(
    vsync: this,
    duration: const Duration(seconds: 11),
  )..repeat();

  @override
  void dispose() {
    _sweep.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return SizedBox.expand(
      child: Stack(
        fit: StackFit.expand,
        children: [
          RepaintBoundary(child: CustomPaint(painter: RadarPainter())),
          RepaintBoundary(child: CustomPaint(painter: SweepPainter(_sweep))),
        ],
      ),
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

    canvas.drawCircle(center, 8, Paint()..color = YeKe.rotom);

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

class SweepPainter extends CustomPainter {
  SweepPainter(this.sweep) : super(repaint: sweep);

  final Animation<double> sweep;

  @override
  void paint(Canvas canvas, Size size) {
    final center = Offset(size.width / 2, size.height / 2);
    final radius = min(size.width, size.height) / 2 - 8;
    final angle = sweep.value * 2 * pi;
    final rect = Rect.fromCircle(center: center, radius: radius);

    final tail = Paint()
      ..shader = SweepGradient(
        colors: [
          YeKe.oddish.withValues(alpha: 0),
          YeKe.oddish.withValues(alpha: 0.45),
        ],
        stops: const [0.75, 1.0],
        transform: GradientRotation(angle),
      ).createShader(rect);

    canvas.drawCircle(center, radius, tail);

    canvas.drawLine(
      center,
      center + Offset(cos(angle), sin(angle) * radius),
      Paint()
        ..color = YeKe.oddish
        ..strokeWidth = 1.5,
    );
  }

  @override
  bool shouldRepaint(covariant SweepPainter oldDelegate) =>
      oldDelegate.sweep != sweep;
}
