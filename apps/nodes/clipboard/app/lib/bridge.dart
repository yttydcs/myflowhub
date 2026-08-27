import 'bridge_contract.dart';
import 'bridge_stub.dart'
    if (dart.library.io) 'bridge_io.dart'
    as implementation;

ClipboardBridge createClipboardBridge() =>
    implementation.createPlatformBridge();
