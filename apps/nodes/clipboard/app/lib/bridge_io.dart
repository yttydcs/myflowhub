import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'bridge_contract.dart';

ClipboardBridge createPlatformBridge() {
  if (Platform.isWindows) {
    return ProcessClipboardBridge();
  }
  return const UnsupportedIOBridge();
}

class ProcessClipboardBridge implements ClipboardBridge {
  Process? _process;
  StreamSubscription<String>? _stdout;
  StreamSubscription<String>? _stderr;
  final Map<String, Completer<dynamic>> _pending =
      <String, Completer<dynamic>>{};
  int _nextID = 0;

  @override
  bool get supported => true;

  @override
  String get platformLabel => 'Windows · local bridge';

  @override
  String get defaultStateDirectory {
    final appData = Platform.environment['APPDATA'];
    if (appData == null || appData.isEmpty) {
      return 'state\\clipboard';
    }
    return '$appData\\MyFlowHub\\ClipboardNode';
  }

  @override
  Future<dynamic> call(String operation, Map<String, dynamic> payload) async {
    await _ensureStarted();
    final id = (++_nextID).toString();
    final completer = Completer<dynamic>();
    _pending[id] = completer;
    final encoded = jsonEncode(<String, dynamic>{
      'version': 1,
      'id': id,
      'op': operation,
      'payload': payload,
    });
    if (utf8.encode(encoded).length > 512 * 1024) {
      _pending.remove(id);
      throw StateError('Clipboard bridge request exceeds 512 KiB.');
    }
    _process!.stdin.writeln(encoded);
    return completer.future.timeout(
      const Duration(seconds: 20),
      onTimeout: () {
        _pending.remove(id);
        throw TimeoutException('Clipboard bridge request timed out.');
      },
    );
  }

  Future<void> _ensureStarted() async {
    if (_process != null) {
      return;
    }
    final configured = Platform.environment['MFH_CLIPBOARD_BACKEND'];
    final executable = configured == null || configured.isEmpty
        ? '${File(Platform.resolvedExecutable).parent.path}\\mfh-clipboard.exe'
        : configured;
    final process = await Process.start(executable, const <String>['-bridge']);
    _process = process;
    _stdout = process.stdout
        .transform(utf8.decoder)
        .transform(const LineSplitter())
        .listen(_handleLine, onError: _failAll);
    _stderr = process.stderr
        .transform(utf8.decoder)
        .listen((_) {}, onError: (_) {});
    unawaited(
      process.exitCode.then((code) {
        _process = null;
        _failAll(StateError('Clipboard backend exited with code $code.'));
      }),
    );
  }

  void _handleLine(String line) {
    if (utf8.encode(line).length > 512 * 1024) {
      _failAll(StateError('Clipboard bridge response exceeds 512 KiB.'));
      return;
    }
    final decoded = jsonDecode(line) as Map<String, dynamic>;
    final id = decoded['id'] as String? ?? '';
    final pending = _pending.remove(id);
    if (pending == null) {
      return;
    }
    if (decoded['ok'] == true) {
      pending.complete(decoded['result']);
      return;
    }
    pending.completeError(
      StateError(decoded['error'] as String? ?? 'Bridge error'),
    );
  }

  void _failAll(Object error) {
    final values = _pending.values.toList(growable: false);
    _pending.clear();
    for (final pending in values) {
      pending.completeError(error);
    }
  }

  @override
  Future<void> close() async {
    final process = _process;
    _process = null;
    if (process != null) {
      process.stdin.close();
      process.kill();
    }
    await _stdout?.cancel();
    await _stderr?.cancel();
    _failAll(StateError('Clipboard bridge closed.'));
  }
}

class UnsupportedIOBridge implements ClipboardBridge {
  const UnsupportedIOBridge();

  @override
  bool get supported => false;

  @override
  String get platformLabel => Platform.operatingSystem;

  @override
  String get defaultStateDirectory => '';

  @override
  Future<dynamic> call(String operation, Map<String, dynamic> payload) async {
    throw UnsupportedError('This platform does not host ClipboardNode.');
  }

  @override
  Future<void> close() async {}
}
