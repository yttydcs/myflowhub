import 'dart:convert';

import 'package:flutter/material.dart';

import 'bridge.dart';
import 'bridge_contract.dart';

void main() => runApp(ClipboardApp(bridge: createClipboardBridge()));

class ClipboardApp extends StatelessWidget {
  const ClipboardApp({super.key, required this.bridge});

  final ClipboardBridge bridge;

  @override
  Widget build(BuildContext context) => MaterialApp(
    title: 'MyFlowHub ClipboardNode',
    debugShowCheckedModeBanner: false,
    theme: ThemeData(
      colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xff4f46e5)),
      scaffoldBackgroundColor: const Color(0xfff6f7fb),
      inputDecorationTheme: const InputDecorationTheme(
        border: OutlineInputBorder(),
        filled: true,
        fillColor: Colors.white,
      ),
      cardTheme: const CardThemeData(
        elevation: 0,
        color: Colors.white,
        margin: EdgeInsets.zero,
      ),
      useMaterial3: true,
    ),
    home: ClipboardHome(bridge: bridge),
  );
}

class ClipboardHome extends StatefulWidget {
  const ClipboardHome({super.key, required this.bridge});

  final ClipboardBridge bridge;

  @override
  State<ClipboardHome> createState() => _ClipboardHomeState();
}

class _ClipboardHomeState extends State<ClipboardHome> {
  late final TextEditingController _state;
  final _node = TextEditingController(text: '11');
  final _parent = TextEditingController(text: '1');
  final _endpoint = TextEditingController(text: '127.0.0.1:7331');
  final _parentKey = TextEditingController();
  final _permit = TextEditingController();
  final _peers = TextEditingController();
  final _send = TextEditingController();

  Map<String, dynamic>? _configuration;
  Map<String, dynamic>? _status;
  List<dynamic> _history = const <dynamic>[];
  bool _running = false;
  bool _busy = false;
  String? _message;

  @override
  void initState() {
    super.initState();
    _state = TextEditingController(text: widget.bridge.defaultStateDirectory);
  }

  @override
  void dispose() {
    widget.bridge.close();
    for (final controller in <TextEditingController>[
      _state,
      _node,
      _parent,
      _endpoint,
      _parentKey,
      _permit,
      _peers,
      _send,
    ]) {
      controller.dispose();
    }
    super.dispose();
  }

  Future<void> _run(Future<void> Function() action) async {
    if (_busy) return;
    setState(() {
      _busy = true;
      _message = null;
    });
    try {
      await action();
    } catch (error) {
      if (mounted) setState(() => _message = error.toString());
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _identity() => _run(() async {
    final result = await widget.bridge.call('identity', <String, dynamic>{
      'version': 1,
      'state_directory': _state.text.trim(),
      'node_id': _node.text.trim(),
    }) as Map<String, dynamic>;
    setState(() {
      _message =
          'Node ${result['node_id']} public key: ${result['public_key']}';
    });
  });

  Future<void> _start() => _run(() async {
    final payload = <String, dynamic>{
      'version': 1,
      'state_directory': _state.text.trim(),
      'node_id': _node.text.trim(),
      'parent_node_id': _parent.text.trim(),
      'endpoint': _endpoint.text.trim(),
      'parent_public_key': _parentKey.text.trim(),
    };
    final permit = _permit.text.trim();
    if (permit.isNotEmpty) payload['permit'] = jsonDecode(permit);
    await widget.bridge.call('start', payload);
    setState(() => _running = true);
    await _refreshUnlocked();
  });

  Future<void> _stop() => _run(() async {
    await widget.bridge.call('stop', <String, dynamic>{'version': 1});
    setState(() {
      _running = false;
      _status = null;
      _configuration = null;
      _history = const <dynamic>[];
    });
  });

  Future<void> _refresh() => _run(_refreshUnlocked);

  Future<void> _refreshUnlocked() async {
    final status = await widget.bridge.call('status', <String, dynamic>{
      'version': 1,
    }) as Map<String, dynamic>;
    final configuration = await widget.bridge.call(
      'configuration',
      <String, dynamic>{'version': 1},
    ) as Map<String, dynamic>;
    final history = await widget.bridge.call('history', <String, dynamic>{
      'version': 1,
    }) as List<dynamic>;
    final peers = configuration['peers'] as List<dynamic>? ?? const [];
    setState(() {
      _running = true;
      _status = status;
      _configuration = configuration;
      _history = history;
      _peers.text = peers
          .map((entry) => (entry as Map<String, dynamic>)['node_id'])
          .join(', ');
    });
  }

  Future<void> _saveConfiguration() => _run(() async {
    final current = _configuration;
    if (current == null) throw StateError('Start the node first.');
    final peerIDs =
        _peers.text
            .split(',')
            .map((value) => value.trim())
            .where((value) => value.isNotEmpty)
            .toList()
          ..sort((left, right) => int.parse(left).compareTo(int.parse(right)));
    final update = <String, dynamic>{
      'version': 1,
      'expected_revision': current['revision'],
      'enabled': current['enabled'],
      'max_inline_bytes': current['max_inline_bytes'],
      'auto_watch': current['auto_watch'],
      'auto_apply': current['auto_apply'],
      'history_retention': current['history_retention'],
      'history_limit': current['history_limit'],
      'history_max_bytes': current['history_max_bytes'],
      'history_ttl_ms': current['history_ttl_ms'],
      'peers': peerIDs
          .map((id) => <String, dynamic>{'node_id': id, 'receive': true})
          .toList(),
    };
    final updated = await widget.bridge.call(
      'configuration.update',
      update,
    ) as Map<String, dynamic>;
    setState(() => _configuration = updated);
    await _refreshUnlocked();
  });

  Future<void> _sendText() => _run(() async {
    await widget.bridge.call('send', <String, dynamic>{
      'version': 1,
      'text': _send.text,
    });
    _send.clear();
    await _refreshUnlocked();
  });

  Future<void> _clearHistory() => _run(() async {
    await widget.bridge.call('history.clear', <String, dynamic>{'version': 1});
    await _refreshUnlocked();
  });

  @override
  Widget build(BuildContext context) {
    final columns = MediaQuery.sizeOf(context).width >= 1000;
    return Scaffold(
      appBar: AppBar(
        title: const Text('MyFlowHub ClipboardNode'),
        actions: <Widget>[
          Padding(
            padding: const EdgeInsets.only(right: 20),
            child: Chip(
              avatar: Icon(
                widget.bridge.supported ? Icons.link : Icons.visibility,
                size: 18,
              ),
              label: Text(widget.bridge.platformLabel),
            ),
          ),
        ],
      ),
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.all(24),
          children: <Widget>[
            if (!widget.bridge.supported)
              const _Notice(
                text: 'This Web build previews configuration. Run the Windows product to host a native node.',
              ),
            if (_message != null) _Notice(text: _message!),
            if (_busy) const LinearProgressIndicator(),
            const SizedBox(height: 16),
            if (columns)
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: <Widget>[
                  Expanded(child: _connectionCard()),
                  const SizedBox(width: 20),
                  Expanded(child: _configurationCard()),
                ],
              )
            else ...<Widget>[
              _connectionCard(),
              const SizedBox(height: 20),
              _configurationCard(),
            ],
            const SizedBox(height: 20),
            _activityCard(),
          ],
        ),
      ),
    );
  }

  Widget _connectionCard() => _Panel(
    title: 'Node connection',
    subtitle: _status == null ? 'Stopped' : 'Runtime state available',
    child: Column(
      children: <Widget>[
        _field(_state, 'State directory'),
        Row(
          children: <Widget>[
            Expanded(child: _field(_node, 'Node ID')),
            const SizedBox(width: 12),
            Expanded(child: _field(_parent, 'Parent ID')),
          ],
        ),
        _field(_endpoint, 'Parent endpoint'),
        _field(_parentKey, 'Parent public key', obscure: true),
        _field(_permit, 'Provisioning permit JSON', lines: 2),
        Wrap(
          spacing: 10,
          runSpacing: 10,
          children: <Widget>[
            OutlinedButton(
              onPressed: _busy ? null : _identity,
              child: const Text('Create identity'),
            ),
            FilledButton(
              onPressed: _busy || _running ? null : _start,
              child: const Text('Start node'),
            ),
            OutlinedButton(
              onPressed: _busy || !_running ? null : _refresh,
              child: const Text('Refresh'),
            ),
            TextButton(
              onPressed: _busy || !_running ? null : _stop,
              child: const Text('Stop'),
            ),
          ],
        ),
      ],
    ),
  );

  Widget _configurationCard() {
    final config = _configuration;
    if (config == null) {
      return const _Panel(
        title: 'Synchronization policy',
        subtitle: 'Start the node to edit its durable configuration.',
        child: SizedBox(
          height: 220,
          child: Center(child: Icon(Icons.tune, size: 48)),
        ),
      );
    }
    return _Panel(
      title: 'Synchronization policy',
      subtitle: 'Revision ${config['revision']} · explicit receive peers',
      child: Column(
        children: <Widget>[
          _switch(config, 'enabled', 'Synchronization enabled'),
          _switch(config, 'auto_watch', 'Watch local clipboard'),
          _switch(config, 'auto_apply', 'Apply remote clipboard automatically'),
          _field(_peers, 'Receive from Node IDs (comma-separated)'),
          Align(
            alignment: Alignment.centerRight,
            child: FilledButton.icon(
              onPressed: _busy ? null : _saveConfiguration,
              icon: const Icon(Icons.save_outlined),
              label: const Text('Save policy'),
            ),
          ),
        ],
      ),
    );
  }

  Widget _switch(Map<String, dynamic> config, String key, String title) =>
      SwitchListTile(
        contentPadding: EdgeInsets.zero,
        title: Text(title),
        value: config[key] as bool,
        onChanged: (value) => setState(() => config[key] = value),
      );

  Widget _activityCard() => _Panel(
    title: 'Clipboard activity',
    subtitle: 'Bodies remain in protected events and local bounded history; status contains metadata only.',
    child: Column(
      children: <Widget>[
        TextField(
          controller: _send,
          minLines: 2,
          maxLines: 5,
          decoration: const InputDecoration(labelText: 'Send text explicitly'),
        ),
        const SizedBox(height: 12),
        Row(
          children: <Widget>[
            FilledButton.icon(
              onPressed: _busy || !_running ? null : _sendText,
              icon: const Icon(Icons.send),
              label: const Text('Send'),
            ),
            const Spacer(),
            TextButton.icon(
              onPressed: _busy || !_running ? null : _clearHistory,
              icon: const Icon(Icons.delete_outline),
              label: const Text('Clear history'),
            ),
          ],
        ),
        const Divider(height: 28),
        if (_history.isEmpty)
          const Padding(
            padding: EdgeInsets.all(24),
            child: Text('No retained clipboard events.'),
          )
        else
          ..._history.take(20).map((entry) {
            final item = entry as Map<String, dynamic>;
            final text = item['text'] as String?;
            return ListTile(
              contentPadding: EdgeInsets.zero,
              leading: const CircleAvatar(
                child: Icon(Icons.content_paste, size: 18),
              ),
              title: Text(
                text == null || text.isEmpty ? 'Body not retained' : text,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
              ),
              subtitle: Text(
                '${item['kind']} · ${item['size_bytes']} bytes · ${item['origin_node_id']}',
              ),
            );
          }),
      ],
    ),
  );

  Widget _field(
    TextEditingController controller,
    String label, {
    int lines = 1,
    bool obscure = false,
  }) => Padding(
    padding: const EdgeInsets.only(bottom: 12),
    child: TextField(
      controller: controller,
      maxLines: lines,
      obscureText: obscure,
      decoration: InputDecoration(labelText: label),
    ),
  );
}

class _Panel extends StatelessWidget {
  const _Panel({
    required this.title,
    required this.subtitle,
    required this.child,
  });

  final String title;
  final String subtitle;
  final Widget child;

  @override
  Widget build(BuildContext context) => Card(
    child: Padding(
      padding: const EdgeInsets.all(20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Text(title, style: Theme.of(context).textTheme.titleLarge),
          const SizedBox(height: 4),
          Text(subtitle, style: Theme.of(context).textTheme.bodySmall),
          const SizedBox(height: 20),
          child,
        ],
      ),
    ),
  );
}

class _Notice extends StatelessWidget {
  const _Notice({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) => Container(
    padding: const EdgeInsets.all(14),
    decoration: BoxDecoration(
      color: Theme.of(context).colorScheme.secondaryContainer,
      borderRadius: BorderRadius.circular(12),
    ),
    child: Text(text),
  );
}
