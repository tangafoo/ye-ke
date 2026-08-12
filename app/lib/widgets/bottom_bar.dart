import 'package:flutter/material.dart';
import '../theme/moth.dart';

class MenuItem {
  final String label;
  final String icon;
  const MenuItem(this.label, this.icon);
}

const _menuItems = <MenuItem>[
  MenuItem('Scenarios', ''),
  MenuItem('Quote', ''),
  MenuItem('Profile', ''),
];

class BottomBar extends StatelessWidget {
  final int index;
  final ValueChanged<int> onSelect;

  const BottomBar({super.key, required this.index, required this.onSelect});

  static const _barHeight = 64.0;
  static const _livestreamBtnHeight = 80.0;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: _livestreamBtnHeight + 8,
      child: Stack(
        children: [
          Positioned(
            left: 0,
            right: 0,
            bottom: 0,
            height: _barHeight,
            child: Container(
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
            ),
          ),
          Positioned(
            left: 16,
            bottom: 16,
            child: GestureDetector(
              onTap: () {},
              child: Container(
                height: _livestreamBtnHeight,
                width: _livestreamBtnHeight,
                padding: const EdgeInsets.all(6),
                decoration: const BoxDecoration(
                  color: Moth.detained,
                  shape: BoxShape.circle,
                ),
                child: Image.asset(
                  'assets/images/camera-icon.webp',
                  height: 20,
                ),
              ),
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
      color: Colors.transparent,
      child: InkWell(
        onTap: onTap,
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
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
