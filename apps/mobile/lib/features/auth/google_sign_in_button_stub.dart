import 'package:flutter/material.dart';

Widget buildGoogleSignInButton({
  required VoidCallback onPressed,
  required bool enabled,
}) {
  return SizedBox(
    height: 52,
    child: OutlinedButton.icon(
      onPressed: enabled ? onPressed : null,
      icon: const Text(
        'G',
        style: TextStyle(
          color: Color(0xFF4285F4),
          fontSize: 20,
          fontWeight: FontWeight.w600,
        ),
      ),
      label: const Text('Tiếp tục với Google'),
    ),
  );
}
