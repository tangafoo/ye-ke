import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../theme/ye_ke.dart';

class AskBar extends StatefulWidget {
  final String hint;
  final bool? useIcon;
  final ValueChanged<String> onSubmit;

  const AskBar({
    super.key,
    required this.hint,
    this.useIcon = false,
    required this.onSubmit,
  });

  @override
  State<AskBar> createState() => _AskBarState();
}

class _AskBarState extends State<AskBar> {
  final _controller = TextEditingController();

  void _submit() {
    final q = _controller.text.trim();
    if (q.isEmpty) return;

    HapticFeedback.lightImpact();
    widget.onSubmit(q);
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        IntrinsicHeight(
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.end,
            children: [
              Expanded(
                child: TextField(
                  minLines: 1,
                  maxLines: 3,
                  controller: _controller,
                  textInputAction: TextInputAction.search,
                  onSubmitted: (_) => _submit(),
                  style: const TextStyle(color: YeKe.night, fontSize: 15),
                  cursorColor: YeKe.detained,
                  decoration: InputDecoration(
                    suffixIcon: ValueListenableBuilder(
                      valueListenable: _controller,
                      builder: (context, value, _) => value.text.isEmpty
                          ? const SizedBox.shrink()
                          : IconButton(
                              icon: const Icon(
                                Icons.close,
                                size: 18,
                                color: YeKe.leather,
                              ),
                              onPressed: _controller.clear,
                            ),
                    ),
                    border: OutlineInputBorder(
                      borderRadius: BorderRadius.only(
                        topLeft: Radius.circular(6),
                        topRight: Radius.circular(6),
                        bottomLeft: Radius.circular(6),
                        bottomRight: Radius.circular(6),
                      ),
                      borderSide: BorderSide.none,
                    ),
                    filled: true,
                    fillColor: YeKe.bg0,
                    hintText: widget.hint,
                    hintStyle: const TextStyle(
                      color: YeKe.leather,
                      fontSize: 15,
                    ),
                  ),
                ),
              ),
              const SizedBox(width: 6),
              GestureDetector(
                onTap: _submit,
                child: Container(
                  height: 50,
                  width: 60,
                  decoration: BoxDecoration(color: YeKe.detained),
                  alignment: Alignment.center,
                  child: widget.useIcon!
                      ? Icon(Icons.message, color: YeKe.bg0)
                      : const Text(
                          "Ask",
                          style: TextStyle(
                            fontSize: 16,
                            fontWeight: FontWeight.bold,
                            fontFamily: 'Faculty',
                          ),
                        ),
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }
}
