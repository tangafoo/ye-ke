import 'dart:ui';
import 'package:byakugan/screens/live_screen.dart';

import 'screens/chat_screen.dart';
import 'screens/location_primer_screen.dart';
import 'screens/map_screen.dart';
import 'screens/profile_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:geolocator/geolocator.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'services/identity_service.dart';
import 'widgets/bottom_bar.dart';

import 'theme/ye_ke.dart';

void main() {
  runApp(const ByakuganApp());
}

const _kPrimerDone = 'location_primer_done';

/// Startup gate: kicks off anonymous registration (fire-and-forget, never
/// blocks launch) and decides whether the location primer runs before the
/// shell. The primer shows once — after that, straight to the app.
class StartupGate extends StatefulWidget {
  const StartupGate({super.key});

  @override
  State<StartupGate> createState() => _StartupGateState();
}

class _StartupGateState extends State<StartupGate> {
  bool? _showPrimer; // null while deciding

  @override
  void initState() {
    super.initState();
    _decide();
  }

  Future<void> _decide() async {
    IdentityService().ensureRegistered();

    final prefs = await SharedPreferences.getInstance();
    var show = false;
    if (!(prefs.getBool(_kPrimerDone) ?? false)) {
      try {
        // `denied` is also the never-asked state — the one the primer is for.
        show = await Geolocator.checkPermission() == LocationPermission.denied;
      } catch (_) {
        show = false; // platform without location support
      }
    }
    if (!mounted) return;
    setState(() => _showPrimer = show);
  }

  Future<void> _primerFinished() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setBool(_kPrimerDone, true);
    if (!mounted) return;
    setState(() => _showPrimer = false);
  }

  @override
  Widget build(BuildContext context) {
    return switch (_showPrimer) {
      null => const ColoredBox(color: YeKe.night),
      true => LocationPrimerScreen(onFinished: _primerFinished),
      false => const AppShell(),
    };
  }
}

class AppShell extends StatefulWidget {
  const AppShell({super.key});

  @override
  State<AppShell> createState() => _AppShellState();
}

class _AppShellState extends State<AppShell> {
  var _index = 0;

  Future<void> _onCamera() async {
    final isDesktop = MediaQuery.sizeOf(context).width > 600;

    final choice = isDesktop
        ? await showDialog<String>(
            context: context,
            builder: (_) => const Dialog(
              backgroundColor: YeKe.bg0,
              child: _CameraActions(),
            ),
          )
        : await showModalBottomSheet<String>(
            context: context,
            backgroundColor: YeKe.bg0,
            builder: (_) => const _CameraActions(),
          );

    if (!mounted) return;
    switch (choice) {
      case 'live':
        Navigator.of(
          context,
        ).push(MaterialPageRoute(builder: (_) => const LiveScreen()));
      case 'report':
      case null:
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      extendBody: true,
      body: Stack(
        children: [
          IndexedStack(
            index: _index,
            children: const [ChatScreen(), MapScreen(), ProfileScreen()],
          ),
          Positioned(
            top: 0,
            right: 0,
            child: SafeArea(
              child: Padding(
                padding: const EdgeInsets.only(top: 12, right: 12),
                child: Image.asset(
                  'assets/images/activity-icon.webp',
                  height: 42,
                ),
              ),
            ),
          ),
        ],
      ),
      floatingActionButton: CameraButton(onTap: () => _onCamera()),
      floatingActionButtonLocation: const CameraBtnLocation(),
      bottomNavigationBar: BottomBar(
        index: _index,
        onSelect: (i) => setState(() => _index = i),
      ),
    );
  }
}

class _CameraActions extends StatelessWidget {
  const _CameraActions();

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            ListTile(
              tileColor: YeKe.detained,
              title: Text(
                'Start Livestream',
                style: TextStyle(color: Colors.white, fontFamily: 'Faculty'),
              ),
              onTap: () => Navigator.pop(context, 'live'),
            ),
          ],
        ),
      ),
    );
  }
}

class ByakuganApp extends StatelessWidget {
  const ByakuganApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Ye Ke',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        brightness: Brightness.dark,
        scaffoldBackgroundColor: YeKe.bg0,
        fontFamily: 'Chelsea',
        useMaterial3: true,
      ),
      home: const StartupGate(),
    );
  }
}

/// A one-tap scenario category. Routes "situation first" — tapping it will
/// later ask "who's in front of you?" (PDRM / JPJ / DBKL / religious).
class Category {
  final String en;
  final String bm;
  final IconData icon;
  final Color accent;

  const Category(this.en, this.bm, this.icon, this.accent);
}

const _categories = <Category>[
  Category(
    'Road stop',
    'Disekat di jalan',
    Icons.directions_car_filled,
    YeKe.road,
  ),
  Category('Detained?', 'Ditahan?', Icons.pan_tool, YeKe.detained),
  Category('At my door', 'Di pintu rumah', Icons.door_front_door, YeKe.door),
  Category(
    'Where I stand',
    'Di mana saya berdiri',
    Icons.local_fire_department,
    YeKe.stand,
  ),
];

class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key});

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  bool _bm = false; // false = EN, true = BM. App will remember this later.

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Stack(
        children: [
          // Cool blue base gradient.
          const _Backdrop(),
          // Soft electric glow rising from the search bar.
          const Positioned(left: 0, right: 0, bottom: -120, child: _Glow()),
          SafeArea(
            child: Padding(
              padding: const EdgeInsets.fromLTRB(20, 12, 20, 20),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  _TopBar(bm: _bm, onToggle: (v) => setState(() => _bm = v)),
                  const SizedBox(height: 18),
                  // Tiles own the whole upper canvas now (no headline).
                  Expanded(
                    child: GridView.count(
                      crossAxisCount: 2,
                      mainAxisSpacing: 14,
                      crossAxisSpacing: 14,
                      childAspectRatio: 0.92,
                      children: [
                        for (final c in _categories)
                          _CategoryTile(category: c, bm: _bm),
                      ],
                    ),
                  ),
                  const SizedBox(height: 14),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _Backdrop extends StatelessWidget {
  const _Backdrop();
  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: const BoxDecoration(
        gradient: LinearGradient(
          begin: Alignment.topCenter,
          end: Alignment.bottomCenter,
          colors: [YeKe.bg1, YeKe.bg1],
        ),
      ),
    );
  }
}

class _Glow extends StatelessWidget {
  const _Glow();
  @override
  Widget build(BuildContext context) {
    return IgnorePointer(
      child: Container(
        height: 360,
        decoration: const BoxDecoration(
          gradient: RadialGradient(
            radius: 0.7,
            colors: [Color(0x552E6BFF), Color(0x00060912)],
          ),
        ),
      ),
    );
  }
}

class _TopBar extends StatelessWidget {
  final bool bm;
  final ValueChanged<bool> onToggle;
  const _TopBar({required this.bm, required this.onToggle});

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Image.asset('assets/images/logo-white.webp', height: 40),
        const Spacer(),
        _LangToggle(bm: bm, onToggle: onToggle),
        const SizedBox(width: 12),
        // Profile stub (no auth yet — mocked).
        Container(
          height: 38,
          width: 38,
          decoration: BoxDecoration(
            color: YeKe.glass,
            shape: BoxShape.circle,
            border: Border.all(color: YeKe.stroke),
          ),
          child: const Icon(Icons.person_outline, size: 20, color: YeKe.sub),
        ),
      ],
    );
  }
}

class _LangToggle extends StatelessWidget {
  final bool bm;
  final ValueChanged<bool> onToggle;
  const _LangToggle({required this.bm, required this.onToggle});

  @override
  Widget build(BuildContext context) {
    return Container(
      height: 38,
      padding: const EdgeInsets.all(3),
      decoration: BoxDecoration(
        color: YeKe.glass,
        borderRadius: BorderRadius.circular(20),
        border: Border.all(color: YeKe.stroke),
      ),
      child: Row(
        children: [
          _segment('BM', bm, () => onToggle(true)),
          _segment('EN', !bm, () => onToggle(false)),
        ],
      ),
    );
  }

  Widget _segment(String label, bool active, VoidCallback onTap) {
    return GestureDetector(
      onTap: () {
        HapticFeedback.selectionClick();
        onTap();
      },
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 160),
        padding: const EdgeInsets.symmetric(horizontal: 14),
        alignment: Alignment.center,
        decoration: BoxDecoration(
          color: active ? YeKe.glow : Colors.transparent,
          borderRadius: BorderRadius.circular(16),
        ),
        child: Text(
          label,
          style: TextStyle(
            color: active ? Colors.white : YeKe.sub,
            fontWeight: FontWeight.w700,
            fontSize: 13,
          ),
        ),
      ),
    );
  }
}

class _CategoryTile extends StatelessWidget {
  final Category category;
  final bool bm;
  const _CategoryTile({required this.category, required this.bm});

  @override
  Widget build(BuildContext context) {
    return ClipRRect(
      borderRadius: BorderRadius.circular(22),
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: 12, sigmaY: 12),
        child: Material(
          color: Colors.transparent,
          child: InkWell(
            onTap: () {
              HapticFeedback.lightImpact();
              ScaffoldMessenger.of(context)
                ..hideCurrentSnackBar()
                ..showSnackBar(
                  SnackBar(
                    behavior: SnackBarBehavior.floating,
                    backgroundColor: YeKe.bg1,
                    content: Text(
                      'Next: "who is in front of you?" → ${category.en}',
                      style: const TextStyle(color: YeKe.text),
                    ),
                  ),
                );
            },
            child: Container(
              padding: const EdgeInsets.all(18),
              decoration: BoxDecoration(
                gradient: LinearGradient(
                  begin: Alignment.topLeft,
                  end: Alignment.bottomRight,
                  colors: [category.accent.withValues(alpha: 0.18), YeKe.glass],
                ),
                borderRadius: BorderRadius.circular(22),
                border: Border.all(color: YeKe.stroke),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Container(
                    height: 46,
                    width: 46,
                    decoration: BoxDecoration(
                      color: category.accent.withValues(alpha: 0.16),
                      borderRadius: BorderRadius.circular(14),
                    ),
                    child: Icon(
                      category.icon,
                      color: category.accent,
                      size: 24,
                    ),
                  ),
                  const Spacer(),
                  Text(
                    bm ? category.bm : category.en,
                    style: const TextStyle(
                      color: YeKe.text,
                      fontSize: 19,
                      fontWeight: FontWeight.w800,
                      letterSpacing: -0.3,
                      height: 1.05,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Container(height: 3, width: 26, color: category.accent),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
