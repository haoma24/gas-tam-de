import 'package:flutter/material.dart';
import 'package:google_sign_in_web/web_only.dart' as web;

Widget buildGoogleSignInButton({
  required VoidCallback onPressed,
  required bool enabled,
}) {
  return IgnorePointer(
    ignoring: !enabled,
    child: Opacity(
      opacity: enabled ? 1 : 0.55,
      child: Center(child: web.renderButton()),
    ),
  );
}
