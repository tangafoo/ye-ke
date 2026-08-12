import 'package:flutter/material.dart';
import '../theme/moth.dart';

class MenuItem {
  final String label;
  final String icon;
  const MenuItem(this.label, this.icon);
}

const _menuItems = <MenuItem>[
  MenuItem('Map', 'assets/images/map-icon.webp'),
  MenuItem('Chat', 'assets/images/chat-icon.webp'),
  MenuItem('User', 'assets/images/user-icon.webp'),
];

const _barHeight = 64.0;
const _livestreamBtnHeight = 80.0;

class BottomBar extends StatelessWidget {
  final int index;
  final ValueChanged<int> onSelect;

  const BottomBar({super.key, required this.index, required this.onSelect});

  @override
  Widget build(BuildContext context) {
    return Container(
      height: _barHeight,
      decoration: const BoxDecoration(
        color: Moth.bg1,
        border: Border(top: BorderSide(color: Moth.detained, width: 4)),
      ),
      child: Row(
        children: [
          for (var i = 0; i < _menuItems.length + 1; i++)
            Expanded(
              child: i == 0
                  ? Container(color: Moth.glass)
                  : _MenuSlot(
                      item: _menuItems[i - 1],
                      selected: index == i - 1,
                      onTap: () => onSelect(i - 1),
                    ),
            ),
        ],
      ),
    );
  }
}

class _MenuSlot extends StatelessWidget {
  final MenuItem item;
  final bool selected;
  final VoidCallback onTap;

  const _MenuSlot({
    required this.item,
    required this.selected,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return Material(
      color: selected ? Moth.glow : Colors.transparent,
      child: InkWell(
        onTap: onTap,
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            if (item.icon.isNotEmpty)
              Expanded(
                child: Padding(
                  padding: const EdgeInsets.all(6),
                  child: Image.asset(item.icon),
                ),
              ),
            const SizedBox(height: 2),
            Text(
              item.label,
              style: TextStyle(
                color: selected ? Moth.text : Moth.sub,
                fontSize: 11,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class CameraButton extends StatelessWidget {
  final VoidCallback onTap;
  const CameraButton({super.key, required this.onTap});

  @override
  Widget build(BuildContext context) {
    return Material(
      color: Moth.detained,
      shape: const CircleBorder(),
      elevation: 6,
      shadowColor: Color(0x73000000),
      child: InkWell(
        onTap: onTap,
        customBorder: const CircleBorder(),
        child: Container(
          height: _livestreamBtnHeight,
          width: _livestreamBtnHeight,
          padding: const EdgeInsets.all(6),
          alignment: Alignment.center,
          child: Image.asset('assets/images/camera-icon.webp'),
        ),
      ),
    );
  }
}

class CameraBtnLocation extends FloatingActionButtonLocation {
  const CameraBtnLocation();

  @override
  Offset getOffset(ScaffoldPrelayoutGeometry scaffoldGeometry) {
    final base = FloatingActionButtonLocation.startDocked.getOffset(
      scaffoldGeometry,
    );
    return base + const Offset(0, 12);
  }
}
