import 'bridge_contract.dart';

ClipboardBridge createPlatformBridge() => const UnsupportedBridge();

class UnsupportedBridge implements ClipboardBridge {
  const UnsupportedBridge();

  @override
  bool get supported => false;

  @override
  String get platformLabel => 'Web preview';

  @override
  String get defaultStateDirectory => '';

  @override
  Future<dynamic> call(String operation, Map<String, dynamic> payload) async {
    throw UnsupportedError(
      'Web builds provide the configuration UI only; a native node runtime is required.',
    );
  }

  @override
  Future<void> close() async {}
}
