import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

import 'auth_session.dart';

const _kSessionKey = 'gas_tam_de.auth_session.v1';

/// Local persistence for [AuthSession] (Web + mobile MVP).
class AuthSessionStore {
  AuthSessionStore(this._prefs);

  final SharedPreferences _prefs;

  Future<AuthSession?> load() async {
    final raw = _prefs.getString(_kSessionKey);
    if (raw == null || raw.isEmpty) return null;
    try {
      final map = jsonDecode(raw);
      if (map is! Map<String, dynamic>) return null;
      return AuthSession.fromJson(map);
    } catch (_) {
      return null;
    }
  }

  Future<void> save(AuthSession session) async {
    await _prefs.setString(_kSessionKey, jsonEncode(session.toJson()));
  }

  Future<void> clear() async {
    await _prefs.remove(_kSessionKey);
  }
}
