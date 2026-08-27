import 'package:flutter_test/flutter_test.dart';

import 'dart:ui';

import 'package:myflowhub_clipboard/bridge_contract.dart';
import 'package:myflowhub_clipboard/main.dart';

class FakeBridge implements ClipboardBridge {
  @override
  String get defaultStateDirectory => 'test-state';

  @override
  String get platformLabel => 'Test bridge';

  @override
  bool get supported => true;

  @override
  Future<dynamic> call(String operation, Map<String, dynamic> payload) async =>
      null;

  @override
  Future<void> close() async {}
}

void main() {
  testWidgets('renders node, policy, and privacy surfaces', (tester) async {
    tester.view.physicalSize = const Size(1400, 1200);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    await tester.pumpWidget(ClipboardApp(bridge: FakeBridge()));
    expect(find.text('MyFlowHub ClipboardNode'), findsOneWidget);
    expect(find.text('Node connection'), findsOneWidget);
    expect(find.text('Synchronization policy'), findsOneWidget);
    expect(find.text('Clipboard activity'), findsOneWidget);
    expect(find.textContaining('metadata only'), findsOneWidget);
  });
}
