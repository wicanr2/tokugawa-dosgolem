"""核對平台修正後的正常輸入與凍結 HD 收據，不把中途停止當完成。"""
import argparse
import hashlib
import json
import os
from pathlib import Path


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    assert Path('/.dockerenv').is_file() and os.getuid() == 1000
    parser = argparse.ArgumentParser()
    parser.add_argument('--revision', choices=['platform', 'rcr'], required=True)
    args = parser.parse_args()
    root, project = Path('/output'), Path('/project')
    revision = args.revision
    binary_dir = root / ('runtime/platform/ready' if revision == 'platform' else 'runtime/rcr')
    test_log = binary_dir / 'startup-probe/platform-test.log'
    log = test_log.read_text()
    assert 'FAIL' not in log and len([line for line in log.splitlines() if line.startswith('ok')]) == 4
    binary_sha = sha(binary_dir / 'tokugawa-desktop-prototype')
    current_hashes = None
    proofs = {}
    for mode, folder, variant in [
        ('title', 'desktop-title', revision),
        ('title-missing', 'desktop-title-missing', 'platform'),
        ('weather', 'desktop-regression', revision),
    ]:
        directory = root / folder / variant
        receipt_path = directory / f'desktop-{mode}.json'
        receipt = json.loads(receipt_path.read_text())
        proof_path = directory / f'verification-{mode}.json'
        proof = json.loads(proof_path.read_text())
        assert proof['verified'] and proof['receipt_sha256'] == sha(receipt_path)
        verifier = 'verify-hd-weather.py' if mode == 'weather' else 'verify-hd-remaining-runtime.py'
        assert proof['tool_sha256'] == sha(project / 'tools' / verifier)
        hashes = receipt['integration_input_sha256']
        for name, digest in hashes.items():
            assert sha(directory / 'source-snapshot' / name) == digest, name
            if variant == revision:
                assert sha(project / name) == digest, name
        if variant == revision:
            assert sha(directory / 'tokugawa-desktop-prototype') == binary_sha
            if current_hashes is None:
                current_hashes = hashes
            else:
                assert hashes == current_hashes
        proofs[mode] = {'revision': variant, 'proof_sha256': sha(proof_path),
                        'receipt_sha256': sha(receipt_path), 'binary_sha256': sha(directory / 'tokugawa-desktop-prototype'),
                        'integration_inputs': hashes}
    if revision == 'rcr':
        old = proofs['title-missing']['integration_inputs']
        differences = sorted(name for name in current_hashes if old[name] != current_hashes[name])
        assert differences == ['tools/cpu386-integration_test.go', 'tools/dosgolem.patch'], differences
    menus = {}
    for name, variant in [('menu-history', 'platform/ready' if revision == 'platform' else 'rcr'),
                          ('menu-exit', 'platform/ready')]:
        directory = root / 'source-boundary' / name / variant
        receipt_path = directory / 'receipt.json'
        receipt = json.loads(receipt_path.read_text())
        for source, digest in receipt['integration_input_sha256'].items():
            assert sha(directory / 'source-snapshot' / source) == digest, source
        events = receipt['mouse_input_events']
        expected_steps = [80000000, 81000000] if name == 'menu-history' else [80000000]
        assert [event['scheduled_step'] for event in events] == expected_steps
        assert [event['actual_step'] for event in events] == expected_steps
        # 離開於放開前自然結束；輸入來自同一份普通滑鼠時間表。
        original_directory = directory.parent.parent if variant == 'platform/ready' else directory.parent
        assert sha(directory / 'mouse.json') == sha(original_directory / 'mouse.json')
        if name == 'menu-exit':
            assert receipt['stop'] == 'dos_exit' and receipt['successful_steps'] == 80017192
            memory = (directory / 'startup-memory.bin').read_bytes()
            assert memory[0x449] == 3 and int.from_bytes(memory[0x44a:0x44c], 'little') == 80
        menus[name] = {'revision': variant, 'receipt_sha256': sha(receipt_path),
                       'successful_steps': receipt['successful_steps'], 'stop': receipt['stop'],
                       'normal_path_completed': receipt['stop'] == 'dos_exit'}
    result = {'verified': True, 'revision': revision, 'binary_sha256': binary_sha,
              'test_log_sha256': sha(test_log), 'tool_sha256': sha(Path(__file__)),
              'gui_proofs': proofs, 'ordinary_menu_probes': menus,
              'limitations': ['menu-history stopping reason is retained; not a completed HD path',
                              'exit proves DOS termination and BDA mode restoration, not text rendering',
                              'other nineteen new art sources, five source tails, PCX and packaging remain incomplete']}
    target = binary_dir / 'verification-summary.json'
    assert not target.exists()
    target.write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps({'verified': True, 'revision': revision, 'summary_sha256': sha(target),
                      'menus': menus}, ensure_ascii=False))


if __name__ == '__main__':
    main()
