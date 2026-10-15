"""彙整捲動開場的實際視窗、完整狀態與首戰回歸。"""
import hashlib
import json
import os
from pathlib import Path

from PIL import Image


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def read(path):
    return json.loads(path.read_text())


def main():
    assert Path('/.dockerenv').is_file() and os.getuid() == 1000
    root, project = Path('/output'), Path('/project')
    assert root.stat().st_uid == 1000
    frozen = read(root/'runtime/frozen-integration-input.json')
    assert len(frozen) == 86
    binary = sha(root/'runtime/tokugawa-desktop-prototype')
    log = root/'runtime/startup-probe/platform-test.log'
    text = log.read_text()
    assert 'FAIL' not in text and sum(line.startswith('ok') for line in text.splitlines()) == 4
    source = root/'source-verification.json'
    assert read(source)['verified'] and sha(source) == 'b46d791603cf03c77261106439f703744edb5f67bceecb86bf60a14f4508e319'
    proofs = {}
    for folder, mode, tool in [('desktop', 'shift', 'verify-hd-opening-shift.py'),
                               ('desktop-missing', 'shift-missing', 'verify-hd-opening-shift.py'),
                               ('regression/ready', 'weather', 'verify-hd-weather.py')]:
        directory = root/folder
        proof_path = directory/f'verification-{mode}.json'
        receipt_path = directory/f'desktop-{mode}.json'
        proof, receipt = read(proof_path), read(receipt_path)
        assert proof['verified'] and proof['tool_sha256'] == sha(project/'tools'/tool)
        assert proof['receipt_sha256'] == sha(receipt_path)
        assert sha(directory/'tokugawa-desktop-prototype') == binary
        assert receipt['integration_input_sha256'] == frozen
        for name, digest in frozen.items():
            assert digest == sha(project/name) == sha(directory/'source-snapshot'/name), (folder, name)
        assert proof['complete_RAM_sha256'] == sha(directory/'startup-memory.bin')
        pictures = []
        if mode != 'weather':
            assert proof['graphics_completed_returns'] == 700 and len(proof['checks']) == 5
            assert [r['source'] for r in proof['checks']] == ['OP2.OZM', 'OP2.OZM', 'OP8.OZM', 'OP8.OZM', 'OP8.OZM']
            assert proof['checks'][2]['unknown_native_pixels']
            for check in proof['checks']:
                trio = check['pictures']
                assert len(trio) == 3
                events = [r['event'] for r in trio]
                assert all(e['paused'] and e['proof'] == events[0]['proof'] and e['step'] == events[0]['step'] for e in events)
                assert events[0]['frame']['png_sha256'] == events[2]['frame']['png_sha256']
                for item in trio:
                    picture = directory/item['actual_png']
                    frame = item['event']['frame']
                    presented = directory/Path(frame['path']).relative_to('/output')
                    assert sha(picture) == item['sha256'] and sha(presented) == frame['png_sha256']
                    assert Image.open(picture).convert('RGBA').tobytes() == Image.open(presented).convert('RGBA').tobytes()
                    native = frame['native_indices']
                    native_path = Path(native['path'])
                    if native_path.is_absolute():
                        native_path = native_path.relative_to('/output')
                    assert sha(directory/native_path) == native['sha256']
                    pictures.append({'path': str(picture.relative_to(root)), 'sha256': sha(picture)})
        else:
            previous = project/'workplace/hd-prototype/catalog/remaining/history/partial/regression'
            for item in proof['pictures']:
                name = item['actual_png']
                assert Image.open(directory/name).convert('RGBA').tobytes() == Image.open(previous/name).convert('RGBA').tobytes()
                pictures.append({'path': str((directory/name).relative_to(root)), 'sha256': sha(directory/name)})
        proofs[mode] = {'path': str(proof_path.relative_to(root)), 'sha256': sha(proof_path),
                        'receipt_sha256': sha(receipt_path), 'complete_RAM_sha256': proof['complete_RAM_sha256'],
                        'pictures': pictures}
    result = {'verified': True, 'classification': 'OP2/OP8 shifted consumers integrated with bounded normal-path X11 samples',
              'tool_sha256': sha(Path(__file__)), 'binary_sha256': binary, 'source_snapshot_count': 86,
              'frozen_inputs_sha256': sha(root/'runtime/frozen-integration-input.json'), 'platform_test_log_sha256': sha(log),
              'source_proof_sha256': sha(source), 'normal_static_sources': ['OP2.OZM', 'OP8.OZM'],
              'implemented_shift_frames': {'OP2.OZM': 241, 'OP8.OZM': 441},
              'normal_X11_samples': 5, 'missing_X11_samples': 5, 'proofs': proofs,
              'limitations': ['unseen shift frames covered by source/shared tests, not individually sampled X11',
                              'four partial-source OZM, other animation consumers, PCX and whole-game HD incomplete',
                              'desktop CD audio, four languages and three-platform packaging remain separately tracked']}
    target = root/'verification-summary.json'
    assert not target.exists()
    target.write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'verified': True, 'sha256': sha(target), 'binary_sha256': binary}))


if __name__ == '__main__':
    main()
