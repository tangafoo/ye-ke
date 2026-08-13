import 'package:camera/camera.dart';
import 'package:flutter/material.dart';
import '../theme/ye_ke.dart';

class LiveScreen extends StatefulWidget {
  const LiveScreen({super.key});

  @override
  State<LiveScreen> createState() => _LiveScreenState();
}

class _LiveScreenState extends State<LiveScreen> {
  CameraController? _controller;
  bool _live = false;

  @override
  void initState() {
    super.initState();
    _setup();
  }

  Future<void> _setup() async {
    final cameras = await availableCameras();
    if (cameras.isEmpty) return;

    final controller = CameraController(cameras.first, ResolutionPreset.high);

    await controller.initialize();
    if (!mounted) return;
    setState(() => _controller = controller);
  }

  @override
  void dispose() {
    _controller?.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final controller = _controller;
    return Scaffold(
      backgroundColor: YeKe.bg1,
      body: Stack(
        fit: StackFit.expand,
        children: [
          if (controller != null && controller.value.isInitialized)
            CameraPreview(controller)
          else
            const Center(
              child: CircularProgressIndicator(color: YeKe.detained),
            ),
          SafeArea(
            child: Column(
              children: [
                Row(
                  children: [
                    IconButton(
                      onPressed: () => Navigator.of(context).pop(),
                      icon: const Icon(
                        Icons.close,
                        color: YeKe.glass,
                        size: 30,
                      ),
                    ),
                    const Spacer(),
                    if (_live)
                      Container(
                        padding: const EdgeInsets.symmetric(
                          horizontal: 10,
                          vertical: 4,
                        ),
                        decoration: BoxDecoration(
                          color: YeKe.detained,
                          borderRadius: BorderRadius.circular(6),
                        ),
                        child: const Text(
                          'LIVE',
                          style: TextStyle(color: YeKe.cement),
                        ),
                      ),
                    const SizedBox(width: 12),
                  ],
                ),
                const Spacer(),
                GestureDetector(
                  onTap: () => setState(() => _live = !_live),
                  child: Container(
                    height: 76,
                    width: 76,
                    margin: const EdgeInsets.only(bottom: 24),
                    decoration: BoxDecoration(
                      shape: BoxShape.circle,
                      border: Border.all(color: YeKe.glass, width: 4),
                      color: _live ? YeKe.detained : YeKe.road,
                    ),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
