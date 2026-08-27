abstract class ClipboardBridge {
  bool get supported;
  String get platformLabel;
  String get defaultStateDirectory;

  Future<dynamic> call(String operation, Map<String, dynamic> payload);
  Future<void> close();
}
