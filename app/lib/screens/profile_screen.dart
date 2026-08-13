import 'package:flutter/material.dart';
import '../theme/ye_ke.dart';

class ProfileScreen extends StatefulWidget {
  const ProfileScreen({super.key});

  @override
  State<ProfileScreen> createState() => _ProfileScreenState();
}

class _ProfileScreenState extends State<ProfileScreen> {
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: YeKe.cement,
      body: Center(
        child: Container(
          margin: const EdgeInsets.symmetric(horizontal: 24),
          width: double.infinity,
          padding: const EdgeInsets.all(20),
          decoration: BoxDecoration(
            boxShadow: [
              BoxShadow(
                blurRadius: 6,
                offset: Offset(0, 6),
                blurStyle: BlurStyle.inner,
              ),
            ],
            color: YeKe.detained,
            borderRadius: BorderRadius.circular(12),
          ),
          child: Column(mainAxisSize: MainAxisSize.min),
        ),
      ),
    );
  }
}
