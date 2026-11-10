import 'dart:math';

import 'package:flutter/material.dart';

import '../theme/ye_ke.dart';

class RadarView extends StatefulWidget {
  const RadarView({
    super.key,
    this.gridColor = YeKe.paint,
    this.sweepColor = YeKe.road,
    this.dotColor = YeKe.rotom,
  });

  final Color gridColor;
  final Color sweepColor;
  final Color dotColor;

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
          RepaintBoundary(
            child: CustomPaint(
              painter: RadarPainter(
                gridColor: widget.gridColor,
                labelColor: widget.sweepColor,
                dotColor: widget.dotColor,
              ),
            ),
          ),
          RepaintBoundary(
            child: CustomPaint(
              painter: SweepPainter(_sweep, color: widget.sweepColor),
            ),
          ),
        ],
      ),
    );
  }
}

class RadarPainter extends CustomPainter {
  RadarPainter({
    required this.gridColor,
    required this.labelColor,
    required this.dotColor,
  });

  static const rings = 3;

  final Color gridColor;
  final Color labelColor;
  final Color dotColor;

  @override
  void paint(Canvas canvas, Size size) {
    final center = Offset(size.width / 2, size.height / 2);
    final radius = min(size.width, size.height) / 2 - 8;

    final line = Paint()
      ..color = gridColor
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

    canvas.drawCircle(center, 8, Paint()..color = dotColor);

    final n = TextPainter(
      text: TextSpan(
        text: 'N',
        style: TextStyle(color: labelColor, fontSize: 10),
      ),
      textDirection: TextDirection.ltr,
    )..layout();

    n.paint(canvas, center + Offset(-n.width / 2, -radius - n.height - 2));
  }

  @override
  bool shouldRepaint(covariant RadarPainter oldDelegate) =>
      oldDelegate.gridColor != gridColor ||
      oldDelegate.labelColor != labelColor ||
      oldDelegate.dotColor != dotColor;
}

class SweepPainter extends CustomPainter {
  SweepPainter(this.sweep, {required this.color}) : super(repaint: sweep);

  final Animation<double> sweep;
  final Color color;

  @override
  void paint(Canvas canvas, Size size) {
    final center = Offset(size.width / 2, size.height / 2);
    final radius = min(size.width, size.height) / 2 - 8;
    final angle = sweep.value * 2 * pi;
    final rect = Rect.fromCircle(center: center, radius: radius);

    final tail = Paint()
      ..shader = SweepGradient(
        colors: [color.withValues(alpha: 0), color.withValues(alpha: 0.45)],
        stops: const [0.75, 1.0],
        transform: GradientRotation(angle),
      ).createShader(rect);

    canvas.drawCircle(center, radius, tail);

    canvas.drawLine(
      center,
      center + Offset(cos(angle), sin(angle)) * radius,
      Paint()
        ..color = color
        ..strokeWidth = 1.5,
    );
  }

  @override
  bool shouldRepaint(covariant SweepPainter oldDelegate) =>
      oldDelegate.sweep != sweep || oldDelegate.color != color;
}
