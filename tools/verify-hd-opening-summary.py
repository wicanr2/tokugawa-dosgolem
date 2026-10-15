"""彙整完整複製開場 HD 的來源、色彩、缺圖及正常路徑證據。"""
import hashlib
import json
import os
from pathlib import Path

from PIL import Image


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    assert Path('/.dockerenv').is_file() and os.getuid() == 1000
    project = Path('/project')
    root = project / 'workplace/hd-prototype/catalog/remaining/history'
    source_path = root / 'source-verification.json'
    palette_path = root / 'palette-effects.json'
    art_path = root / 'art-v2/profiles.json'
    for path in [source_path, palette_path]:
        assert json.loads(path.read_text())['verified']
    art = json.loads(art_path.read_text())
    assert art['tool_sha256'] == sha(project / 'tools/hd-opening-prepare.py')
    assert art['record']['status'] == 'runtime_observed'
    assert art['record']['source_proof_sha256'] == sha(source_path)
    assert art['record']['asset_sha256'] == sha(project / 'assets/hd-b/art' / art['record']['asset'])
    binary_path = root / 'runtime/palette/tokugawa-desktop-prototype'
    binary_sha = sha(binary_path)
    log_path = root / 'runtime/palette/startup-probe/platform-test.log'
    log = log_path.read_text()
    assert 'FAIL' not in log and sum(line.startswith('ok') for line in log.splitlines()) == 4
    proofs = {}
    source_hashes = None
    for mode, directory, tool in [
        ('opening', 'desktop/palette', 'verify-hd-opening.py'),
        ('opening-missing', 'desktop-missing/palette', 'verify-hd-opening.py'),
        ('opening-effects', 'effects/ready', 'verify-hd-opening.py'),
        ('title', 'title/palette', 'verify-hd-remaining-runtime.py'),
        ('weather', 'regression/palette', 'verify-hd-weather.py'),
    ]:
        folder = root / directory
        proof_path = folder / f'verification-{mode}.json'
        receipt_path = folder / f'desktop-{mode}.json'
        proof = json.loads(proof_path.read_text())
        receipt = json.loads(receipt_path.read_text())
        assert proof['verified'] and proof['receipt_sha256'] == sha(receipt_path)
        if mode.startswith('opening'):
            assert proof['tool_sha256'] == sha(folder / 'verifier-source.txt')
        else:
            assert proof['tool_sha256'] == sha(project / 'tools' / tool)
        assert sha(folder / 'tokugawa-desktop-prototype') == binary_sha
        hashes = receipt['integration_input_sha256']
        assert len(hashes) == 82
        if source_hashes is None:
            source_hashes = hashes
        assert hashes == source_hashes
        for name, digest in hashes.items():
            assert digest == sha(folder / 'source-snapshot' / name) == sha(project / name), name
        loaded = receipt['desktop']['assets']['graphics']
        assert len(loaded) == 29
        for name, info in loaded.items():
            expected = 'fallback_missing_asset' if mode == 'opening-missing' and name == 'OP-FULL.OZM' else 'ready'
            assert info['validation'] == expected, (mode, name)
        toggles = [event for event in receipt['desktop']['events'] if event['kind'] == 'toggle']
        assert all(event['guest_unchanged'] and event['before'] == event['after'] for event in toggles)
        if mode.startswith('opening'):
            assert proof['source_proof_sha256'] == sha(source_path)
            assert proof['palette_contract_sha256'] == sha(palette_path)
            assert proof['art_profiles_sha256'] == sha(art_path)
            assert sha(folder / 'verifier-source.txt') == proof['tool_sha256']
            assert len(proof['stages']) == (8 if mode == 'opening-effects' else 2)
        proofs[mode] = {'path': str(proof_path.relative_to(project)), 'sha256': sha(proof_path),
                        'receipt_sha256': sha(receipt_path), 'complete_RAM_sha256': proof['complete_RAM_sha256'],
                        'toggle_count': len(toggles)}
    effects = json.loads((root / 'effects/ready/verification-opening-effects.json').read_text())['stages']
    assert sum(stage['palette_effect']['kind'] == 'clamp' for stage in effects.values()) == 6
    assert sum(stage['palette_effect']['kind'] == 'cycle' for stage in effects.values()) == 2
    assert all(stage['mode'] == 'hd' and stage['changed_pixels'] > 0 for stage in effects.values())
    missing = json.loads((root / 'desktop-missing/palette/verification-opening-missing.json').read_text())['stages']
    assert missing['OP4.OZM']['mode'] == 'hd'
    assert missing['OP-FULL.OZM']['mode'] == 'fallback_missing_asset'
    assert missing['OP-FULL.OZM']['changed_pixels'] == 0
    previous = project / 'workplace/hd-prototype/catalog/remaining/desktop-regression/rcr'
    weather = json.loads((root / 'regression/palette/verification-weather.json').read_text())
    for picture in weather['pictures']:
        name = picture['actual_png']
        assert Image.open(root / 'regression/palette' / name).convert('RGBA').tobytes() == Image.open(previous / name).convert('RGBA').tobytes()
    result = {'verified': True, 'classification': 'normal complete-copy opening HD and source-specific colour effects',
              'tool_sha256': sha(Path(__file__)), 'binary_sha256': binary_sha,
              'source_proof_sha256': sha(source_path), 'palette_contract_sha256': sha(palette_path),
              'art_profiles_sha256': sha(art_path), 'platform_test_log_sha256': sha(log_path),
              'matched_integration_source_count': 82, 'normal_new_sources': ['OP4.OZM', 'OP-FULL.OZM'],
              'normal_effect_samples': {'clamp': 6, 'cycle': 2}, 'proofs': proofs,
              'HD_colour_transfer': 'presentation approximation; guest palette and timing unchanged',
              'limitations': ['partial and scrolling opening wrappers not yet HD',
                              'five static OZM decode tails remain unsupported; OP-FULL uses exact runtime source',
                              'PCX, all-game HD and three-platform packages remain incomplete']}
    target = root / 'verification-summary.json'
    assert not target.exists()
    target.write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps({'verified': True, 'summary_sha256': sha(target), 'binary_sha256': binary_sha}))


if __name__ == '__main__':
    main()
